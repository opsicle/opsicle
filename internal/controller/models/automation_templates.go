package models

import (
	"database/sql"
	"fmt"
	"opsicle/internal/automations"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type SubmitAutomationTemplateV1Opts struct {
	Db       *sql.DB
	Template automations.Template
	UserId   string
}

type AutomationTemplate struct {
	Id            *string
	Description   *string
	Name          *string
	Version       *int64
	Content       []byte
	Users         []AutomationTemplateUser
	CreatedAt     time.Time
	CreatedBy     User
	LastUpdatedAt time.Time
	LastUpdatedBy User
}

type AutomationTemplateUser struct {
	UserId     string
	TemplateId string
	CanView    bool
	CanExecute bool
	CanUpdate  bool
	CanDelete  bool
	CanInvite  bool
}

func SubmitAutomationTemplateV1(opts SubmitAutomationTemplateV1Opts) (*AutomationTemplate, error) {
	automationTemplate, err := GetAutomationTemplateV1(GetAutomationTemplateV1Opts{
		Db:           opts.Db,
		TemplateName: opts.Template.GetName(),
		UserId:       opts.UserId,
	})
	if err != nil {
		if isMysqlNotFoundError(err) {
			return createAutomationTemplateV1(createAutomationTemplateV1Opts{
				Db:       opts.Db,
				Template: opts.Template,
				UserId:   opts.UserId,
			})
		}
		return nil, err
	}
	return updateAutomationTemplateV1(updateAutomationTemplateV1Opts{
		Db:              opts.Db,
		CurrentTemplate: automationTemplate,
		UpdatedTemplate: opts.Template,
		UserId:          opts.UserId,
	})
}

type updateAutomationTemplateV1Opts struct {
	Db              *sql.DB
	CurrentTemplate *AutomationTemplate
	UpdatedTemplate automations.Template
	UserId          string
}

func updateAutomationTemplateV1(opts updateAutomationTemplateV1Opts) (*AutomationTemplate, error) {
	if opts.CurrentTemplate == nil {
		return nil, fmt.Errorf("empty current template: %w", errorInputValidationFailed)
	}

	description := opts.UpdatedTemplate.GetDescription()
	version := *opts.CurrentTemplate.Version + 1
	updatedTemplateData, err := yaml.Marshal(opts.UpdatedTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal template: %w", err)
	}

	output := AutomationTemplate{
		Id:            opts.CurrentTemplate.Id,
		Name:          opts.CurrentTemplate.Name,
		Description:   &description,
		Version:       &version,
		Content:       updatedTemplateData,
		LastUpdatedAt: time.Now(),
		LastUpdatedBy: User{Id: &opts.UserId},
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
			*output.Id,
			*output.Version,
			string(output.Content),
			opts.UserId,
		},
		FnSource:     "models.updateAutomationTemplateV1[automation_template_versions]",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
	}

	if err := executeMysqlUpdate(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			UPDATE automation_templates
				SET 
					version = ?,
					description = ?,
					last_updated_by = ?
				WHERE 
					id = ?
		`,
		Args: []any{
			*output.Version,
			*output.Description,
			output.LastUpdatedBy.GetId(),
			*output.Id,
		},
		FnSource:     "models.updateAutomationTemplateV1[automation_template_versions]",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
	}

	return &output, nil
}

type createAutomationTemplateV1Opts struct {
	Db       *sql.DB
	Template automations.Template
	UserId   string
}

func createAutomationTemplateV1(opts createAutomationTemplateV1Opts) (*AutomationTemplate, error) {
	automationTemplateUuid := uuid.NewString()

	description := opts.Template.GetDescription()
	name := opts.Template.GetName()
	version := int64(1)
	templateData, err := yaml.Marshal(opts.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal template: %w", err)
	}

	output := AutomationTemplate{
		Id:          &automationTemplateUuid,
		Description: &description,
		Name:        &name,
		Version:     &version,
		Content:     templateData,
		Users: []AutomationTemplateUser{
			{
				UserId:     opts.UserId,
				CanView:    true,
				CanExecute: true,
				CanUpdate:  true,
				CanDelete:  true,
				CanInvite:  true,
			},
		},
	}

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
			*output.Id,
			*output.Name,
			*output.Description,
			*output.Version,
			output.Users[0].UserId,
		},
		FnSource:     "models.createAutomationTemplateV1[automation_templates]",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
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
			*output.Id,
			*output.Version,
			string(output.Content),
			output.Users[0].UserId,
		},
		FnSource:     "models.createAutomationTemplateV1[automation_template_versions]",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
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
			*output.Id,
			output.Users[0].UserId,
			output.Users[0].CanView,
			output.Users[0].CanExecute,
			output.Users[0].CanUpdate,
			output.Users[0].CanDelete,
			output.Users[0].CanInvite,
		},
		FnSource:     "models.createAutomationTemplateV1[automation_template_users]",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
	}

	return &output, nil
}
