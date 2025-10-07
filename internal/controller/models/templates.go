package models

import (
	"database/sql"
	"errors"
	"fmt"
	"opsicle/internal/automations"
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
	Orgs          []TemplateOrg
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

func (t *Template) assertVersion() error {
	if t.Version == nil {
		return fmt.Errorf("%w: missing version", ErrorVersionRequired)
	}
	return nil
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

type AddUserToTemplateV1 struct {
	Db *sql.DB

	UserId     string
	CanView    bool
	CanExecute bool
	CanUpdate  bool
	CanDelete  bool
	CanInvite  bool
	CreatedBy  string
}

func (t *Template) AddUserV1(opts AddUserToTemplateV1) error {
	if err := t.validate(); err != nil {
		return err
	}
	inserts := map[string]any{
		"template_id": t.GetId(),
		"user_id":     opts.UserId,
		"can_view":    opts.CanView,
		"can_execute": opts.CanExecute,
		"can_update":  opts.CanUpdate,
		"can_delete":  opts.CanDelete,
		"can_invite":  opts.CanInvite,
		"created_by":  opts.CreatedBy,
	}
	var insertFields, insertPlaceholders []string
	var insertValues []any
	for field, value := range inserts {
		insertFields = append(insertFields, field)
		insertPlaceholders = append(insertPlaceholders, "?")
		insertValues = append(insertValues, value)
	}
	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(`INSERT INTO template_users(%s) VALUES (%s)`,
			strings.Join(insertFields, ","),
			strings.Join(insertPlaceholders, ","),
		),
		Args:         insertValues,
		FnSource:     "models.Template.AddUserV1",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return err
	}
	return nil
}

func (t *Template) GetId() string {
	return *t.Id
}

func (t *Template) GetName() string {
	return *t.Name
}

func (t *Template) GetVersion() int64 {
	return *t.Version
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
		"template_id",
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
		INSERT INTO template_user_invitations(%s)
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

func (t *Template) ListUsersV1(opts DatabaseConnection) error {
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
				template_users atu
				JOIN users u ON u.id = atu.user_id
				JOIN templates t ON t.id = atu.template_id
			WHERE
				atu.template_id = ?
		`,
		Args: []any{
			*t.Id,
		},
		FnSource: "models.AutomationTemplate.ListUsersV1",
		ProcessRows: func(r *sql.Rows) error {
			user := TemplateUser{
				CreatedBy:     &User{},
				LastUpdatedBy: &User{},
			}
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
				&user.CreatedBy.Id,
				&user.LastUpdatedAt,
				&user.LastUpdatedBy.Id,
			); err != nil {
				return err
			}
			if user.CreatedBy.Id != nil {
				if err := user.CreatedBy.LoadByIdV1(opts); err != nil {
					return fmt.Errorf("failed to load createdBy: %w", err)
				}
			}
			if user.LastUpdatedBy.Id != nil {
				if err := user.LastUpdatedBy.LoadByIdV1(opts); err != nil {
					return fmt.Errorf("failed to load lastUpdatedBy: %w", err)
				}
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
				FROM templates at
				JOIN template_versions atv ON
					atv.template_id = at.id
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
	fieldNames, fieldsToSet, sqlArgs, err := parseUpdateMap(opts.FieldsToSet)
	if err != nil {
		return fmt.Errorf("failed to parse update map: %w", err)
	}
	return executeMysqlUpdate(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(`
			UPDATE templates
				SET %s
				WHERE id = ?
			`,
			strings.Join(fieldsToSet, ", "),
		),
		Args: append(sqlArgs, t.GetId()),
		FnSource: fmt.Sprintf(
			"models.Template.UpdateFieldsV1['%s']",
			strings.Join(fieldNames, "','"),
		),
	})
}

type CreateTemplateVersionV1Opts struct {
	Db       *sql.DB
	Template automations.Template
	UserId   string
}

func CreateTemplateVersionV1(opts CreateTemplateVersionV1Opts) (*Template, error) {
	templateName := opts.Template.GetName()
	automationTemplate, err := GetTemplateV1(GetTemplateV1Opts{
		Db:           opts.Db,
		TemplateName: &templateName,
		UserId:       opts.UserId,
	})
	if err != nil {
		if isMysqlNotFoundError(err) {
			return createTemplate(createTemplateOpts{
				Db:       opts.Db,
				Template: opts.Template,
				UserId:   opts.UserId,
			})
		}
		return nil, err
	}
	return createTemplateVersionV1(createTemplateVersionV1Opts{
		Db:              opts.Db,
		CurrentTemplate: automationTemplate,
		UpdatedTemplate: opts.Template,
		UserId:          opts.UserId,
	})
}

type SubmitOrgTemplateV1Opts struct {
	Db       *sql.DB
	OrgId    string
	Template automations.Template
	UserId   string
}

func SubmitOrgTemplateV1(opts SubmitOrgTemplateV1Opts) (*Template, error) {
	if opts.OrgId == "" {
		return nil, fmt.Errorf("missing organization id: %w", errorInputValidationFailed)
	}
	templateName := opts.Template.GetName()
	orgTemplate, err := GetOrgTemplateV1(GetOrgTemplateV1Opts{
		Db:           opts.Db,
		OrgId:        opts.OrgId,
		TemplateName: &templateName,
		UserId:       opts.UserId,
	})
	if err != nil {
		if isMysqlNotFoundError(err) {
			createdTemplate, createErr := createTemplate(createTemplateOpts{
				Db:       opts.Db,
				Template: opts.Template,
				UserId:   opts.UserId,
				OrgId:    &opts.OrgId,
			})
			if createErr != nil {
				return nil, createErr
			}
			if err := ensureTemplateOrgLink(opts.Db, createdTemplate.GetId(), opts.OrgId, opts.UserId); err != nil {
				return nil, err
			}
			return createdTemplate, nil
		}
		return nil, err
	}
	updatedTemplate, err := createTemplateVersionV1(createTemplateVersionV1Opts{
		Db:              opts.Db,
		CurrentTemplate: orgTemplate,
		UpdatedTemplate: opts.Template,
		UserId:          opts.UserId,
	})
	if err != nil {
		return nil, err
	}
	if err := ensureTemplateOrgLink(opts.Db, updatedTemplate.GetId(), opts.OrgId, opts.UserId); err != nil {
		return nil, err
	}
	return updatedTemplate, nil
}

type createTemplateVersionV1Opts struct {
	Db              *sql.DB
	CurrentTemplate *Template
	UpdatedTemplate automations.Template
	UserId          string
}

func createTemplateVersionV1(opts createTemplateVersionV1Opts) (*Template, error) {
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
			INSERT INTO template_versions (
				template_id,
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
		FnSource:     "models.createTemplateVersionV1[template_versions]",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
	}

	if err := executeMysqlUpdate(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			UPDATE templates
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
		FnSource:     "models.createTemplateVersionV1[template_versions]",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
	}

	return &output, nil
}

func ensureTemplateOrgLink(db *sql.DB, templateId string, orgId string, userId string) error {
	linkId := uuid.NewString()
	insertErr := executeMysqlInsert(mysqlQueryInput{
		Db: db,
		Stmt: `
			INSERT INTO template_orgs (
				id,
				template_id,
				org_id,
				created_by,
				last_updated_by
			) VALUES (
				?,
				?,
				?,
				?,
				?
			)
		`,
		Args: []any{
			linkId,
			templateId,
			orgId,
			userId,
			userId,
		},
		FnSource:     "models.ensureTemplateOrgLink[insert]",
		RowsAffected: oneRowAffected,
	})
	if insertErr != nil {
		if errors.Is(insertErr, ErrorDuplicateEntry) {
			if err := executeMysqlUpdate(mysqlQueryInput{
				Db: db,
				Stmt: `
					UPDATE template_orgs
					SET last_updated_by = ?
					WHERE template_id = ? AND org_id = ?
				`,
				Args: []any{
					userId,
					templateId,
					orgId,
				},
				FnSource: "models.ensureTemplateOrgLink[update]",
			}); err != nil {
				return err
			}
		} else {
			return insertErr
		}
	}
	if err := executeMysqlUpdate(mysqlQueryInput{
		Db: db,
		Stmt: `
			UPDATE templates
				SET org_id = ?
			WHERE id = ?
		`,
		Args: []any{
			orgId,
			templateId,
		},
		FnSource: "models.ensureTemplateOrgLink[templates]",
	}); err != nil {
		return err
	}
	return nil
}

type createTemplateOpts struct {
	Db       *sql.DB
	Template automations.Template
	UserId   string
	OrgId    *string
}

func createTemplate(opts createTemplateOpts) (*Template, error) {
	automationTemplateUuid := uuid.NewString()

	description := opts.Template.GetDescription()
	name := opts.Template.GetName()
	version := int64(1)
	templateData, err := yaml.Marshal(opts.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal template: %w", err)
	}
	orgs := []TemplateOrg{}
	if opts.OrgId != nil {
		orgs = append(orgs, TemplateOrg{OrgId: opts.OrgId})
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
		Orgs: orgs,
	}

	templateInsertMap := map[string]any{
		"id":          *output.Id,
		"name":        *output.Name,
		"description": *output.Description,
		"version":     *output.Version,
		"created_by":  output.Users[0].UserId,
	}
	fn, fv, fvp, err := parseInsertMap(templateInsertMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template insert map: %w", err)
	}
	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(`
			INSERT INTO templates (%s) VALUES (%s)`,
			strings.Join(fn, ", "),
			strings.Join(fvp, ", "),
		),
		Args:         fv,
		FnSource:     "models.createTemplate[templates]",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
	}

	templateVersionInsertMap := map[string]any{
		"template_id": *output.Id,
		"version":     *output.Version,
		"content":     string(output.Content),
		"created_by":  output.Users[0].UserId,
	}
	fn, fv, fvp, err = parseInsertMap(templateVersionInsertMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templateVersion insert map: %w", err)
	}
	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(`
			INSERT INTO template_versions (%s) VALUES (%s)
			`,
			strings.Join(fn, ", "),
			strings.Join(fvp, ", "),
		),
		Args:         fv,
		FnSource:     "models.createTemplate[template_versions]",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
	}

	if len(output.Orgs) > 0 {
		templateOrgInsertMap := map[string]any{
			"id":              uuid.NewString(),
			"template_id":     *output.Id,
			"org_id":          output.Orgs[0].OrgId,
			"created_by":      output.Users[0].UserId,
			"last_updated_by": output.Users[0].UserId,
		}
		fn, fv, fvp, err = parseInsertMap(templateOrgInsertMap)
		if err != nil {
			return nil, fmt.Errorf("failed to parse templateOrg insert map: %w", err)
		}
		if err := executeMysqlInsert(mysqlQueryInput{
			Db: opts.Db,
			Stmt: fmt.Sprintf(`
				INSERT INTO template_orgs (%s) VALUES (%s)`,
				strings.Join(fn, ", "),
				strings.Join(fvp, ", "),
			),
			Args:         fv,
			FnSource:     "models.createTemplate[templateorgs]",
			RowsAffected: oneRowAffected,
		}); err != nil {
			return nil, err
		}
	} else if len(output.Users) > 0 {
		templateUserInsertMap := map[string]any{
			"template_id": *output.Id,
			"user_id":     output.Users[0].UserId,
			"can_view":    output.Users[0].CanView,
			"can_execute": output.Users[0].CanExecute,
			"can_update":  output.Users[0].CanUpdate,
			"can_delete":  output.Users[0].CanDelete,
			"can_invite":  output.Users[0].CanInvite,
		}
		fn, fv, fvp, err = parseInsertMap(templateUserInsertMap)
		if err != nil {
			return nil, fmt.Errorf("failed to parse templateUser insert map: %w", err)
		}
		if err := executeMysqlInsert(mysqlQueryInput{
			Db: opts.Db,
			Stmt: fmt.Sprintf(`
				INSERT INTO template_users (%s) VALUES (%s)`,
				strings.Join(fn, ", "),
				strings.Join(fvp, ", "),
			),
			Args:         fv,
			FnSource:     "models.createTemplate[template_users]",
			RowsAffected: oneRowAffected,
		}); err != nil {
			return nil, err
		}
	}

	return &output, nil
}
