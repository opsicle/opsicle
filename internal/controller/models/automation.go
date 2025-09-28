package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"opsicle/internal/automations"
	"opsicle/internal/queue"
	"opsicle/internal/validate"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func CreateAutomationV1(input *Automation, opts DatabaseConnection) (*Automation, error) {
	if err := validate.Uuid(input.TemplateId); err != nil {
		return nil, fmt.Errorf("%w: %w", errorInputValidationFailed, err)
	} else if input.TemplateVersion <= 0 {
		return nil, fmt.Errorf("%w: missing template version", errorInputValidationFailed)
	} else if len(input.TemplateContent) == 0 {
		return nil, fmt.Errorf("%w: missing template content", errorInputValidationFailed)
	} else if input.TriggeredBy == nil || input.TriggeredBy.Id == nil {
		return nil, fmt.Errorf("%w: missing user", errorInputValidationFailed)
	}
	automationId := uuid.NewString()
	insertMap := map[string]any{
		"id":                automationId,
		"org_id":            input.OrgId,
		"template_content":  input.TemplateContent,
		"template_id":       input.TemplateId,
		"template_version":  input.TemplateVersion,
		"triggered_by":      input.TriggeredBy.GetId(),
		"triggerer_comment": input.TriggererComment,
	}
	fields := []string{}
	valuePlaceholders := []string{}
	values := []any{}
	for field, value := range insertMap {
		fields = append(fields, field)
		valuePlaceholders = append(valuePlaceholders, "?")
		values = append(values, value)
	}
	if err := executeMysqlInsert(mysqlQueryInput{
		Db:       opts.Db,
		FnSource: "models.CreateAutomationV1",
		Stmt: fmt.Sprintf(
			`INSERT INTO automations (%s) VALUES (%s)`,
			strings.Join(fields, ", "),
			strings.Join(valuePlaceholders, ", "),
		),
		Args:         values,
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
	}
	output := input
	output.Id = &automationId
	return output, nil
}

type Automation struct {
	Id                *string
	OrgId             *string
	TemplateId        string
	TemplateVersion   int64
	TemplateContent   []byte
	TemplateCreatedBy *User
	TriggeredBy       *User
	TriggeredAt       time.Time
	TriggererComment  string
}

func (a *Automation) assertId() error {
	if a.Id == nil {
		return fmt.Errorf("%w: missing id", ErrorIdRequired)
	} else if err := validate.Uuid(*a.Id); err != nil {
		return fmt.Errorf("%w: invalid id", err)
	}
	return nil
}

func (a *Automation) assertTemplateContent() error {
	if len(a.TemplateContent) == 0 {
		return fmt.Errorf("%w: missing template content", ErrorTemplateContentRequired)
	}
	return nil
}

func (a *Automation) GetTemplate() (*automations.Template, error) {
	return automations.LoadAutomationTemplate(a.TemplateContent)
}

func (a *Automation) Load(opts DatabaseConnection) error {
	if a.TriggeredBy == nil {
		a.TriggeredBy = &User{}
	}
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT
				id,
				org_id,
				template_content,
				template_id,
				template_version,
				triggered_at,
				triggered_by,
				triggerer_comment
				FROM automations
					WHERE id = ?
		`,
		Args:     []any{*a.Id},
		FnSource: fmt.Sprintf("models.Automation.Load[%s]", *a.Id),
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(
				&a.Id,
				&a.OrgId,
				&a.TemplateContent,
				&a.TemplateId,
				&a.TemplateVersion,
				&a.TriggeredAt,
				&a.TriggeredBy.Id,
				&a.TriggererComment,
			)
		},
	}); err != nil {
		return err
	}
	return nil
}

type QueueAutomationRunV1Output struct {
	AutomationRunId    string
	AutomationQueuedAt time.Time
}

type QueueAutomationRunV1Opts struct {
	Db    *sql.DB
	Input map[string]any
	Q     queue.Queue
}

// QueueRun submits the automation to the queue for processing
func (a *Automation) QueueRunV1(opts QueueAutomationRunV1Opts) (*QueueAutomationRunV1Output, error) {
	if err := a.assertId(); err != nil {
		return nil, err
	} else if err = a.assertTemplateContent(); err != nil {
		return nil, err
	}
	template, err := a.GetTemplate()
	if err != nil {
		return nil, fmt.Errorf("invalid template content: %w", err)
	}

	variables := template.GetVariables()
	finalVariableMap := map[string]any{}
	errs := []error{}

	fmt.Println("!-------------------------------------------------------")
	fmt.Println("-------------------------------------------------------")
	o, _ := json.MarshalIndent(opts.Input, "", "  ")
	fmt.Println(string(o))
	fmt.Println("-------------------------------------------------------")
	fmt.Println("!-------------------------------------------------------")
	for variableId, variable := range variables {
		inputValue, inputVarExists := opts.Input[variableId]
		// assign defaults if it exists
		if !inputVarExists {
			if variable.Default != nil {
				logrus.Infof("assigned variable[%s] the default value[%v]", variableId, variable.Default)
				inputValue = variable.Default
			}
		}
		// if it's marked as required and doesn't have a value yet...
		if variable.IsRequired && inputValue == nil {
			errs = append(errs, fmt.Errorf("var[%s] is required", variableId))
			continue
		}
		// verify the type matches
		switch variable.Type {
		case "bool":
			if _, ok := inputValue.(bool); !ok {
				errs = append(errs, fmt.Errorf("var[%s] should be but is not a boolean", variableId))
				continue
			}
		case "string":
			if _, ok := inputValue.(string); !ok {
				errs = append(errs, fmt.Errorf("var[%s] should be but is not a string", variableId))
				continue
			}
		case "number":
			if _, ok := inputValue.(float64); !ok {
				errs = append(errs, fmt.Errorf("var[%s] should be but is not a number", variableId))
				continue
			}
		case "float":
			if _, ok := inputValue.(float64); !ok {
				errs = append(errs, fmt.Errorf("var[%s] should be but is not a floating point", variableId))
				continue
			}
		}
		finalVariableMap[variableId] = inputValue
	}

	if len(errs) > 0 {
		fmt.Println("yes it reached here")
		return nil, errors.Join(errs...)
	}
	fmt.Println("aisdjioasjdioajsiodjaiosdjioasjiodjasiodjas")

	for i, variable := range template.Spec.Variables {
		template.Spec.Variables[i].Value = finalVariableMap[variable.Id]
	}
	automation := template.Spec.Template
	automation.Variables = template.Spec.Variables
	automation.Status.Id = uuid.NewString()
	automation.Status.QueuedAt = time.Now()
	automationData, err := json.MarshalIndent(automation, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("invalid automation format: %w", errorInputValidationFailed)
	}

	variableData, err := json.MarshalIndent(finalVariableMap, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("invalid variables format: %w", errorInputValidationFailed)
	}

	if err := executeMysqlInsert(mysqlQueryInput{
		FnSource: "models.Automation.QueueRunV1",
		Db:       opts.Db,
		Stmt: `INSERT INTO automation_runs (
			automation_id,
			input_vars,
			last_known_status
		) VALUES (?, ?, ?)`,
		Args: []any{
			*a.Id,
			string(variableData),
			"pending",
		},
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, fmt.Errorf("failed to insert automation run to db: %w", err)
	}

	_, err = opts.Q.Push(queue.PushOpts{
		Data: automationData,
		Queue: queue.QueueOpts{
			Stream:  "automations",
			Subject: "runs",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert automation run to q: %w", err)
	}

	output := &QueueAutomationRunV1Output{
		AutomationRunId:    automation.Status.Id,
		AutomationQueuedAt: automation.Status.QueuedAt,
	}

	return output, nil
}
