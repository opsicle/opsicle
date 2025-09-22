package models

import (
	"fmt"
	"opsicle/internal/automations"
	"opsicle/internal/validate"
	"strings"
	"time"

	"github.com/google/uuid"
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

func (a *Automation) GetTemplate() (*automations.Template, error) {
	return automations.LoadAutomationTemplate(a.TemplateContent)
}
