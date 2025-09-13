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
	TemplateId    *string
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

func (tu *TemplateUser) LoadV1(opts DatabaseConnection) error {
	if err := tu.validate(); err != nil {
		return fmt.Errorf("failed to validate TemplateUser: %w", err)
	}
	return executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
		  SELECT 
				can_view,
				can_execute,
				can_update,
				can_delete,
				can_invite,
				created_at,
				created_by,
				last_updated_at,
				last_updated_by
			FROM
				automation_template_users
			WHERE
				automation_template_id = ?
				AND user_id = ?
		`,
		Args: []any{
			*tu.TemplateId,
			*tu.UserId,
		},
		FnSource: "models.TemplateUser.LoadV1",
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(
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
