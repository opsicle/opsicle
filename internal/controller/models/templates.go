package models

import (
	"database/sql"
	"errors"
	"fmt"
	"opsicle/internal/automations"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type AutomationTemplate struct {
	Id            *string
	Description   *string
	Name          *string
	Version       *int64
	Content       []byte
	Users         []AutomationTemplateUser
	Versions      []TemplateVersion
	CreatedAt     time.Time
	CreatedBy     User
	LastUpdatedAt *time.Time
	LastUpdatedBy *User
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

type TemplateVersion struct {
	AutomationTemplateId string
	Version              int64
	Content              string
	CreatedAt            time.Time
	CreatedBy            User
}

func (t *AutomationTemplate) GetId() string {
	return *t.Id
}

func (t *AutomationTemplate) LoadUsersV1(opts DatabaseConnection) error {
	if t.Id == nil {
		return fmt.Errorf("%w: template id not specified", ErrorInvalidInput)
	}
	users := []AutomationTemplateUser{}
	if err := executeMysqlSelects(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
		  SELECT 
				user_id,
				can_view,
				can_execute,
				can_update,
				can_delete,
				can_invite
			FROM
				automation_template_users
			WHERE
				automation_template_id = ?
		`,
		Args: []any{
			*t.Id,
		},
		FnSource: "models.AutomationTemplate.LoadV1",
		ProcessRows: func(r *sql.Rows) error {
			user := AutomationTemplateUser{TemplateId: t.GetId()}
			if err := r.Scan(
				&user.UserId,
				&user.CanView,
				&user.CanExecute,
				&user.CanUpdate,
				&user.CanDelete,
				&user.CanInvite,
			); err != nil {
				return err
			}
			users = append(users, user)
			return nil
		},
	}); err != nil {
		return err
	}
	t.Users = users
	return nil
}

func (t *AutomationTemplate) LoadV1(opts DatabaseConnection) error {
	if t.Id == nil {
		return fmt.Errorf("%w: template id not specified", ErrorInvalidInput)
	}

	t.CreatedBy = User{}
	t.LastUpdatedBy = &User{}
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
		  SELECT 
				at.id,
				at.name,
				at.description,
				at.created_at,
				at.created_by,
				at.last_updated_at,
				at.last_updated_by,
				atv.content,
				atv.version
				FROM automation_templates at
				JOIN automation_template_versions atv ON
					atv.automation_template_id = at.id
					AND atv.version = at.version
			WHERE
				at.id = ?
		`,
		Args: []any{
			*t.Id,
		},
		FnSource: "models.AutomationTemplate.LoadV1",
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(
				&t.Id,
				&t.Name,
				&t.Description,
				&t.CreatedAt,
				&t.CreatedBy.Id,
				&t.LastUpdatedAt,
				&t.LastUpdatedBy.Id,
				&t.Content,
				&t.Version,
			)
		},
	}); err != nil {
		return err
	}
	if t.CreatedBy.Id != nil {
		if err := t.CreatedBy.LoadByIdV1(opts); err != nil {
			return fmt.Errorf("failed to load createdBy: %w", err)
		}
	}
	if t.LastUpdatedBy.Id != nil {
		if err := t.LastUpdatedBy.LoadByIdV1(opts); err != nil {
			return fmt.Errorf("failed to load createdBy: %w", err)
		}
	}
	return nil
}

func (t *AutomationTemplate) UpdateFieldsV1(opts UpdateFieldsV1) error {
	if err := t.validate(); err != nil {
		return err
	}
	sqlArgs := []any{}
	fieldNames := []string{}
	fieldsToSet := []string{}
	for field, value := range opts.FieldsToSet {
		fieldNames = append(fieldNames, field)
		switch v := value.(type) {
		case string, int, int32, int64, float32, float64, bool:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, v)
		case []byte:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, string(v))
		case DatabaseFunction:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = %s", field, v))
		default:
			valueType := reflect.TypeOf(v)
			return fmt.Errorf("field[%s] has invalid type '%s'", field, valueType.String())
		}
	}
	return executeMysqlUpdate(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(`
			UPDATE automation_templates
				SET %s
				WHERE id = ?
			`,
			strings.Join(fieldsToSet, ", "),
		),
		Args: append(sqlArgs, t.GetId()),
		FnSource: fmt.Sprintf(
			"models.AutomationTemplate.UpdateFieldsV1['%s']",
			strings.Join(fieldNames, "','"),
		),
	})
}

func (t *AutomationTemplate) validate() error {
	errs := []error{}
	if t.Id == nil || *t.Id == "" {
		errs = append(errs, fmt.Errorf("%w: missing id", ErrorIdRequired))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

type SubmitAutomationTemplateV1Opts struct {
	Db       *sql.DB
	Template automations.Template
	UserId   string
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
	return submitAutomationTemplateV1(submitAutomationTemplateV1Opts{
		Db:              opts.Db,
		CurrentTemplate: automationTemplate,
		UpdatedTemplate: opts.Template,
		UserId:          opts.UserId,
	})
}

type submitAutomationTemplateV1Opts struct {
	Db              *sql.DB
	CurrentTemplate *AutomationTemplate
	UpdatedTemplate automations.Template
	UserId          string
}

func submitAutomationTemplateV1(opts submitAutomationTemplateV1Opts) (*AutomationTemplate, error) {
	if opts.CurrentTemplate == nil {
		return nil, fmt.Errorf("empty current template: %w", errorInputValidationFailed)
	}

	description := opts.UpdatedTemplate.GetDescription()
	version := *opts.CurrentTemplate.Version + 1
	updatedTemplateData, err := yaml.Marshal(opts.UpdatedTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal template: %w", err)
	}

	timeNow := time.Now()
	output := AutomationTemplate{
		Id:            opts.CurrentTemplate.Id,
		Name:          opts.CurrentTemplate.Name,
		Description:   &description,
		Version:       &version,
		Content:       updatedTemplateData,
		LastUpdatedAt: &timeNow,
		LastUpdatedBy: &User{Id: &opts.UserId},
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
		FnSource:     "models.submitAutomationTemplateV1[automation_template_versions]",
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
		FnSource:     "models.submitAutomationTemplateV1[automation_template_versions]",
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
