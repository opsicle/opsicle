package models

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TemplateUserInvitation struct {
	Id            string     `json:"id" yaml:"id"`
	InviterId     string     `json:"inviterId" yaml:"inviterId"`
	InviterEmail  *string    `json:"inviterEmail" yaml:"inviterEmail"`
	AcceptorId    *string    `json:"acceptorId,omitempty" yaml:"acceptorId,omitempty"`
	AcceptorEmail *string    `json:"acceptorEmail,omitempty" yaml:"acceptorEmail,omitempty"`
	TemplateId    string     `json:"templateId" yaml:"templateId"`
	TemplateName  *string    `json:"templateName" yaml:"templateName"`
	JoinCode      string     `json:"joinCode" yaml:"joinCode"`
	CreatedAt     time.Time  `json:"createdAt" yaml:"createdAt"`
	LastUpdatedAt *time.Time `json:"lastUpdatedAt" yaml:"lastUpdatedAt"`
	TemplateUser
}

func (tui TemplateUserInvitation) assertAcceptorIdDefined() error {
	if tui.AcceptorId == nil {
		return fmt.Errorf("acceptor id missing: %w", ErrorInvalidInput)
	} else if _, err := uuid.Parse(*tui.AcceptorId); err != nil {
		return fmt.Errorf("acceptor id invalid: %w", ErrorInvalidInput)
	}
	return nil
}

func (tui TemplateUserInvitation) assertIdDefined() error {
	if tui.Id == "" {
		return fmt.Errorf("invitation id missing: %w", ErrorInvalidInput)
	} else if _, err := uuid.Parse(tui.Id); err != nil {
		return fmt.Errorf("invitation id invalid: %w", ErrorInvalidInput)
	}
	return nil
}

func (tui *TemplateUserInvitation) DeleteByIdV1(opts DatabaseConnection) error {
	if err := tui.assertIdDefined(); err != nil {
		return err
	}
	if err := executeMysqlDelete(mysqlQueryInput{
		Db:           opts.Db,
		Stmt:         "DELETE FROM template_user_invitations WHERE id = ?",
		Args:         []any{tui.Id},
		RowsAffected: oneRowAffected,
		FnSource:     "models.TemplateUserInvitation.DeleteByIdV1",
	}); err != nil {
		return err
	}
	return nil
}

func (tui *TemplateUserInvitation) LoadV1(opts DatabaseConnection) error {
	if err := tui.assertIdDefined(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT 
				tui.inviter_id,
				tui.acceptor_id,
				tui.acceptor_email,
				tui.template_id,
				tui.join_code,
				tui.can_view,
				tui.can_execute,
				tui.can_update,
				tui.can_delete,
				tui.can_invite,
				tui.created_at,
				tui.last_updated_at
				FROM template_user_invitations tui
				WHERE id = ?
		`,
		Args:     []any{tui.Id},
		FnSource: "models.TemplateUserInvitation.LoadV1",
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(
				&tui.InviterId,
				&tui.AcceptorId,
				&tui.AcceptorEmail,
				&tui.TemplateId,
				&tui.JoinCode,
				&tui.CanView,
				&tui.CanExecute,
				&tui.CanUpdate,
				&tui.CanDelete,
				&tui.CanInvite,
				&tui.CreatedAt,
				&tui.LastUpdatedAt,
			)
		},
	})
}

func (tui *TemplateUserInvitation) ReplaceAcceptorEmailWithId(opts DatabaseConnection) error {
	if err := tui.assertAcceptorIdDefined(); err != nil {
		return err
	} else if err := tui.assertIdDefined(); err != nil {
		return err
	}
	if err := executeMysqlUpdate(mysqlQueryInput{
		Db:           opts.Db,
		Stmt:         `UPDATE template_user_invitations SET acceptor_id = ? WHERE id = ?`,
		Args:         []any{*tui.AcceptorId, tui.Id},
		RowsAffected: oneRowAffected,
		FnSource:     "models.TemplateUserInvitation.ReplaceAcceptorEmailWithId",
	}); err != nil {
		return err
	}
	return nil
}
