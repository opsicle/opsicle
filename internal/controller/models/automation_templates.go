package models

import (
	"database/sql"
	"fmt"
	"opsicle/internal/automations"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type CreateAutomationTemplateV1Opts struct {
	Db       *sql.DB
	Template automations.Template
	UserId   string
}

func CreateAutomationTemplateV1(opts CreateAutomationTemplateV1Opts) (string, error) {
	automationTemplateUuid := uuid.NewString()

	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			INSERT INTO automation_templates (
				id,
				name,
				description,
				version,
				created_by
			) VALUES (
				?, 
				?, 
				?, 
				?, 
				?
			)
		`,
		Args: []any{
			automationTemplateUuid,
			opts.Template.GetName(),
			opts.Template.GetDescription(),
			1,
			opts.UserId,
		},
		FnSource:     "models.CreateAutomationTemplateV1[automation_templates]",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return "", err
	}

	templateData, err := yaml.Marshal(opts.Template)
	if err != nil {
		return "", fmt.Errorf("failed to marshal template: %w", err)
	}

	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			INSERT INTO automation_template_versions (
				automation_template_id,
				version,
				content,
				created_by
			) VALUES (
			 ?,
			 ?,
			 ?,
			 ?
			)
		`,
		Args: []any{
			automationTemplateUuid,
			1,
			string(templateData),
			opts.UserId,
		},
		FnSource:     "models.CreateAutomationTemplateV1[automation_template_versions]",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return "", err
	}

	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			INSERT INTO automation_template_users (
				automation_template_id,
				user_id,
				can_view,
				can_execute,
				can_update,
				can_delete,
				can_invite
			) VALUES (
			  ?,
			  ?,
			  ?,
			  ?,
			  ?,
			  ?,
			  ?
			)
		`,
		Args: []any{
			automationTemplateUuid,
			opts.UserId,
			true,
			true,
			true,
			true,
			true,
		},
		FnSource:     "models.CreateAutomationTemplateV1[automation_template_users]",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return "", err
	}

	return automationTemplateUuid, nil
}
