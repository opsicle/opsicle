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

type Template struct {
	Id            *string
	Description   *string
	Name          *string
	Version       *int64
	Content       []byte
	Users         []TemplateUser
	Versions      []TemplateVersion
	CreatedAt     time.Time
	CreatedBy     *User
	LastUpdatedAt *time.Time
	LastUpdatedBy *User
}

type TemplateVersion struct {
	AutomationTemplateId string
	Version              int64
	Content              string
	CreatedAt            time.Time
	CreatedBy            User
}

func (t *Template) validate() error {
	errs := []error{}
	if t.Id == nil {
		return fmt.Errorf("%w: missing template id", ErrorIdRequired)
	} else if _, err := uuid.Parse(*t.Id); err != nil {
		return fmt.Errorf("%w: invalid template id", ErrorInvalidInput)
	}
	if len(errs) > 0 {
		errs = append(errs, errorInputValidationFailed)
		return errors.Join(errs...)
	}
	return nil
}

func (t *Template) GetId() string {
	return *t.Id
}

func (t *Template) GetName() string {
	return *t.Name
}

type InviteTemplateUserV1Opts struct {
	Db *sql.DB

	AcceptorId    *string
	AcceptorEmail *string
	InviterId     string
	JoinCode      string
	CanView       bool
	CanExecute    bool
	CanUpdate     bool
	CanDelete     bool
	CanInvite     bool
}

func (t *Template) InviteUserV1(opts InviteTemplateUserV1Opts) (*InviteUserV1Output, error) {
	invitationId := uuid.NewString()

	sqlInserts := []string{
		"id",
		"automation_template_id",
		"inviter_id",
		"join_code",
		"can_view",
		"can_execute",
		"can_update",
		"can_delete",
		"can_invite",
	}
	sqlArgs := []any{
		invitationId,
		t.GetId(),
		opts.InviterId,
		opts.JoinCode,
		opts.CanView,
		opts.CanExecute,
		opts.CanUpdate,
		opts.CanDelete,
		opts.CanInvite,
	}
	isExistingUser := false
	if opts.AcceptorId != nil {
		sqlInserts = append(sqlInserts, "acceptor_id")
		sqlArgs = append(sqlArgs, *opts.AcceptorId)
		isExistingUser = true
	} else if opts.AcceptorEmail != nil {
		sqlInserts = append(sqlInserts, "acceptor_email")
		sqlArgs = append(sqlArgs, *opts.AcceptorEmail)
	} else {
		return nil, fmt.Errorf("failed to receive either acceptor email or id: %w", ErrorInvalidInput)
	}
	sqlPlaceholders := []string{}
	for range len(sqlInserts) {
		sqlPlaceholders = append(sqlPlaceholders, "?")
	}
	sqlStmt := fmt.Sprintf(`
		INSERT INTO automation_template_user_invitations(%s)
			VALUES (%s)
			ON DUPLICATE KEY UPDATE
				can_view = VALUES(can_view),
				can_delete = VALUES(can_delete),
				can_update = VALUES(can_update),
				can_execute = VALUES(can_execute),
				can_invite = VALUES(can_invite),
				last_updated_at = NOW()
	`,
		strings.Join(sqlInserts, ", "),
		strings.Join(sqlPlaceholders, ", "),
	)

	if err := executeMysqlInsert(mysqlQueryInput{
		Db:           opts.Db,
		Stmt:         sqlStmt,
		Args:         sqlArgs,
		FnSource:     "models.Template.InviteUserV1",
		RowsAffected: atMostNRowsAffected(2),
	}); err != nil {
		return nil, err
	}

	return &InviteUserV1Output{
		InvitationId:   invitationId,
		IsExistingUser: isExistingUser,
	}, nil
}

func (t *Template) LoadUsersV1(opts DatabaseConnection) error {
	if err := t.validate(); err != nil {
		return err
	}
	users := []TemplateUser{}
	if err := executeMysqlSelects(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
		  SELECT 
				t.id,
				t.name,
				u.id,
				u.email,
				atu.can_view,
				atu.can_execute,
				atu.can_update,
				atu.can_delete,
				atu.can_invite,
				atu.created_at,
				atu.created_by,
				atu.last_updated_at,
				atu.last_updated_by
			FROM
				automation_template_users atu
				JOIN users u ON u.id = atu.user_id
				JOIN automation_templates t ON t.id = atu.automation_template_id
			WHERE
				atu.automation_template_id = ?
		`,
		Args: []any{
			*t.Id,
		},
		FnSource: "models.AutomationTemplate.LoadUsersV1",
		ProcessRows: func(r *sql.Rows) error {
			user := TemplateUser{}
			if err := r.Scan(
				&user.TemplateId,
				&user.TemplateName,
				&user.UserId,
				&user.UserEmail,
				&user.CanView,
				&user.CanExecute,
				&user.CanUpdate,
				&user.CanDelete,
				&user.CanInvite,
				&user.CreatedAt,
				&user.CreatedBy,
				&user.LastUpdatedAt,
				&user.LastUpdatedBy,
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

func (t *Template) LoadV1(opts DatabaseConnection) error {
	if err := t.validate(); err != nil {
		return err
	}
	t.CreatedBy = &User{}
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

func (t *Template) UpdateFieldsV1(opts UpdateFieldsV1) error {
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

type SubmitTemplateV1Opts struct {
	Db       *sql.DB
	Template automations.Template
	UserId   string
}

func SubmitTemplateV1(opts SubmitTemplateV1Opts) (*Template, error) {
	automationTemplate, err := GetTemplateV1(GetTemplateV1Opts{
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
	return submitTemplateV1(submitTemplateV1Opts{
		Db:              opts.Db,
		CurrentTemplate: automationTemplate,
		UpdatedTemplate: opts.Template,
		UserId:          opts.UserId,
	})
}

type submitTemplateV1Opts struct {
	Db              *sql.DB
	CurrentTemplate *Template
	UpdatedTemplate automations.Template
	UserId          string
}

func submitTemplateV1(opts submitTemplateV1Opts) (*Template, error) {
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
	output := Template{
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
		FnSource:     "models.submitTemplateV1[automation_template_versions]",
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
		FnSource:     "models.submitTemplateV1[automation_template_versions]",
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

func createAutomationTemplateV1(opts createAutomationTemplateV1Opts) (*Template, error) {
	automationTemplateUuid := uuid.NewString()

	description := opts.Template.GetDescription()
	name := opts.Template.GetName()
	version := int64(1)
	templateData, err := yaml.Marshal(opts.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal template: %w", err)
	}

	output := Template{
		Id:          &automationTemplateUuid,
		Description: &description,
		Name:        &name,
		Version:     &version,
		Content:     templateData,
		Users: []TemplateUser{
			{
				UserId:     &opts.UserId,
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
