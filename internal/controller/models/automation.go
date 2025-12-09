package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"opsicle/internal/automations"
	"opsicle/internal/cache"
	"opsicle/internal/queue"
	"opsicle/internal/validate"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type CreatePendingAutomationV1Opts struct {
	Cache            cache.Cache
	OrgId            *string
	TemplateContent  []byte
	TemplateId       string
	TemplateVersion  int64
	TriggeredBy      string
	TriggererComment string
}

// CreatePendingAutomationV1 creates an Automation instance and inserts
// it into the cache for retrieval after the user confirms the execution
func CreatePendingAutomationV1(opts CreatePendingAutomationV1Opts) (*Automation, error) {
	automationId := uuid.NewString()
	automationInstance := &Automation{
		Id:              &automationId,
		TemplateContent: opts.TemplateContent,
		TemplateId:      opts.TemplateId,
		TemplateVersion: opts.TemplateVersion,
		TriggeredBy: &User{
			Id: &opts.TriggeredBy,
		},
		TriggeredAt:      time.Now(),
		TriggererComment: opts.TriggererComment,
	}
	if opts.OrgId != nil {
		if err := validate.Uuid(*opts.OrgId); err != nil {
			return nil, fmt.Errorf("invalid org id: %w", err)
		}
		automationInstance.OrgId = opts.OrgId
	}
	cacheKey := strings.Join([]string{cachePrefixAutomationPending, automationId}, ":")
	cacheData, err := json.Marshal(automationInstance)
	if err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}
	cacheExpiryDuration := time.Hour * 24 * 7
	if err := opts.Cache.Set(cacheKey, string(cacheData), cacheExpiryDuration); err != nil {
		return nil, fmt.Errorf("models.CreatePendingAutomationV1: failed to update cache: %w", err)
	}
	return automationInstance, nil
}

type Automation struct {
	Id                *string   `json:"id"`
	InputVars         []byte    `json:"inputVars"`
	LastKnownStatus   string    `json:"lastKnownStatus"`
	Logs              []byte    `json:"logs"`
	OrgId             *string   `json:"orgId"`
	TemplateId        string    `json:"templateId"`
	TemplateVersion   int64     `json:"templateVersion"`
	TemplateContent   []byte    `json:"templateContent"`
	TemplateCreatedBy *User     `json:"templateCreatedBy"`
	TriggeredBy       *User     `json:"triggeredBy"`
	TriggeredAt       time.Time `json:"triggeredAt"`
	TriggererComment  string    `json:"triggererComment"`
	CreatedAt         time.Time `json:"createdAt"`
	LastUpdatedAt     time.Time `json:"lastUpdatedAt"`
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

func (a *Automation) CreateV1(opts DatabaseConnection) error {
	insertMap := map[string]any{
		"id":                a.Id,
		"input_vars":        a.InputVars,
		"org_id":            a.OrgId,
		"template_content":  a.TemplateContent,
		"template_id":       a.TemplateId,
		"template_version":  a.TemplateVersion,
		"triggered_by":      a.TriggeredBy.GetId(),
		"triggered_at":      a.TriggeredAt,
		"triggerer_comment": a.TriggererComment,
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
		FnSource: "models.Automation.CreateV1",
		Stmt: fmt.Sprintf(
			`INSERT INTO automations (%s) VALUES (%s)`,
			strings.Join(fields, ", "),
			strings.Join(valuePlaceholders, ", "),
		),
		Args:         values,
		RowsAffected: oneRowAffected,
	}); err != nil {
		return err
	}
	return nil
}

func (a *Automation) GetTemplate() (*automations.Template, error) {
	return automations.LoadAutomationTemplate(a.TemplateContent)
}

func (a *Automation) LoadPendingV1(opts DatabaseConnection) error {
	if a.TriggeredBy == nil {
		a.TriggeredBy = &User{}
	}
	cacheKey := strings.Join([]string{cachePrefixAutomationPending, *a.Id}, ":")
	cacheData, err := cache.Get().Get(cacheKey)
	if err != nil {
		return fmt.Errorf("models.Automation.LoadPendingV1: failed to get cache: %w", err)
	}
	if err := json.Unmarshal([]byte(cacheData), a); err != nil {
		return fmt.Errorf("models.Automation.LoadPendingV1: failed to unmarshal data: %w", err)
	}
	return nil
}

func (a *Automation) LoadV1(opts DatabaseConnection) error {
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
	Db *sql.DB
	Q  queue.Instance

	Input map[string]any
	OrgId *string
}

// QueueRun submits the automation to the queue for processing
func (a *Automation) RunV1(opts QueueAutomationRunV1Opts) (*QueueAutomationRunV1Output, error) {
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
		return nil, errors.Join(errs...)
	}

	for i, variable := range template.Spec.Variables {
		template.Spec.Variables[i].Value = finalVariableMap[variable.Id]
	}
	automationSpec := template.Spec.Template
	automationSpec.Variables = template.Spec.Variables
	automationSpec.Status.Id = *a.Id
	automationSpec.Status.QueuedAt = time.Now()
	automationData, err := json.MarshalIndent(automationSpec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("invalid automation format: %w", errorInputValidationFailed)
	}

	a.InputVars, err = json.Marshal(finalVariableMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal variables: %w", err)
	}
	if err := a.CreateV1(DatabaseConnection{Db: opts.Db}); err != nil {
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
		AutomationRunId:    automationSpec.Status.Id,
		AutomationQueuedAt: automationSpec.Status.QueuedAt,
	}

	return output, nil
}
