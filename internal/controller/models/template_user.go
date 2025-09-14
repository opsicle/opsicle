package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func NewTemplateUser(userId, templateId string) TemplateUser {
	return TemplateUser{
		UserId:     &userId,
		TemplateId: &templateId,
	}
}

type TemplateUser struct {
	UserId        *string
	UserEmail     *string
	TemplateId    *string
	TemplateName  *string
	CanView       bool
	CanExecute    bool
	CanUpdate     bool
	CanDelete     bool
	CanInvite     bool
	CreatedAt     time.Time
	CreatedBy     *string
	LastUpdatedAt time.Time
	LastUpdatedBy *string
}

func (tu *TemplateUser) validate() error {
	errs := []error{}
	if tu.UserId == nil {
		errs = append(errs, fmt.Errorf("%w: missing user id", ErrorIdRequired))
	} else if _, err := uuid.Parse(*tu.UserId); err != nil {
		errs = append(errs, fmt.Errorf("%w: invalid user id", ErrorInvalidInput))
	}
	if tu.TemplateId == nil {
		errs = append(errs, fmt.Errorf("%w: missing template id", ErrorIdRequired))
	} else if _, err := uuid.Parse(*tu.TemplateId); err != nil {
		errs = append(errs, fmt.Errorf("%w: invalid template id", ErrorInvalidInput))
	}
	if len(errs) > 0 {
		errs = append(errs, errorInputValidationFailed)
		return errors.Join(errs...)
	}
	return nil
}

func (tu *TemplateUser) DeleteV1(opts DatabaseConnection) error {
	if err := tu.validate(); err != nil {
		return fmt.Errorf("failed to validate TemplateUser: %w", err)
	}
	return executeMysqlDelete(mysqlQueryInput{
		Db:           opts.Db,
		Stmt:         `DELETE FROM automation_template_users WHERE automation_template_id = ? AND user_id = ?`,
		Args:         []any{tu.GetTemplateId(), tu.GetUserId()},
		FnSource:     "models.TemplateUser.DeleteV1",
		RowsAffected: oneRowAffected,
	})
}

func (tu TemplateUser) GetUserEmail() string {
	return *tu.UserEmail
}

func (tu TemplateUser) GetUserId() string {
	return *tu.UserId
}

func (tu TemplateUser) GetTemplateId() string {
	return *tu.TemplateId
}

func (tu TemplateUser) GetTemplateName() string {
	return *tu.TemplateName
}

func (tu *TemplateUser) LoadV1(opts DatabaseConnection) error {
	if err := tu.validate(); err != nil {
		return fmt.Errorf("failed to validate TemplateUser: %w", err)
	}
	return executeMysqlSelect(mysqlQueryInput{
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
				AND atu.user_id = ?
		`,
		Args: []any{
			*tu.TemplateId,
			*tu.UserId,
		},
		FnSource: "models.TemplateUser.LoadV1",
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(
				&tu.TemplateId,
				&tu.TemplateName,
				&tu.UserId,
				&tu.UserEmail,
				&tu.CanView,
				&tu.CanExecute,
				&tu.CanUpdate,
				&tu.CanDelete,
				&tu.CanInvite,
				&tu.CreatedAt,
				&tu.CreatedBy,
				&tu.LastUpdatedAt,
				&tu.LastUpdatedBy,
			)
		},
	})
}
