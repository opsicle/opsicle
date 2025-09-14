package models

import (
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

func (tui *TemplateUserInvitation) ReplaceAcceptorEmailWithId(opts DatabaseConnection) error {
	if err := tui.assertAcceptorIdDefined(); err != nil {
		return err
	} else if err := tui.assertIdDefined(); err != nil {
		return err
	}
	if err := executeMysqlUpdate(mysqlQueryInput{
		Db:           opts.Db,
		Stmt:         `UPDATE automation_template_user_invitations SET acceptor_id = ? WHERE id = ?`,
		Args:         []any{*tui.AcceptorId, tui.Id},
		RowsAffected: oneRowAffected,
		FnSource:     "models.TemplateUserInvitation.ReplaceAcceptorEmailWithId",
	}); err != nil {
		return err
	}
	return nil
}
