package models

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OrgUserInvitation struct {
	Id            string     `json:"id" yaml:"id"`
	InviterId     string     `json:"inviterId" yaml:"inviterId"`
	InviterEmail  *string    `json:"inviterEmail" yaml:"inviterEmail"`
	AcceptorId    *string    `json:"acceptorId,omitempty" yaml:"acceptorId,omitempty"`
	AcceptorEmail *string    `json:"acceptorEmail,omitempty" yaml:"acceptorEmail,omitempty"`
	OrgId         string     `json:"orgId" yaml:"orgId"`
	OrgName       *string    `json:"orgName" yaml:"orgName"`
	OrgCode       *string    `json:"orgCode" yaml:"orgCode"`
	JoinCode      string     `json:"joinCode" yaml:"joinCode"`
	Type          string     `json:"type" yaml:"type"`
	CreatedAt     time.Time  `json:"createdAt" yaml:"createdAt"`
	LastUpdatedAt *time.Time `json:"lastUpdatedAt" yaml:"lastUpdatedAt"`
}

func (oui OrgUserInvitation) assertAcceptorIdDefined() error {
	if oui.AcceptorId == nil {
		return fmt.Errorf("invitation id missing: %w", ErrorInvalidInput)
	} else if _, err := uuid.Parse(*oui.AcceptorId); err != nil {
		return fmt.Errorf("invidation id invalid: %w", ErrorInvalidInput)
	}
	return nil
}

func (oui OrgUserInvitation) assertIdDefined() error {
	if oui.Id == "" {
		return fmt.Errorf("invitation id missing: %w", ErrorInvalidInput)
	} else if _, err := uuid.Parse(oui.Id); err != nil {
		return fmt.Errorf("invidation id invalid: %w", ErrorInvalidInput)
	}
	return nil
}

func (oui *OrgUserInvitation) DeleteByIdV1(opts DatabaseConnection) error {
	if err := oui.assertIdDefined(); err != nil {
		return err
	}
	if err := executeMysqlDelete(mysqlQueryInput{
		Db:           opts.Db,
		Stmt:         "DELETE FROM org_user_invitations WHERE id = ?",
		Args:         []any{oui.Id},
		RowsAffected: oneRowAffected,
		FnSource:     "models.OrgUserInvitation.DeleteByIdV1",
	}); err != nil {
		return err
	}
	return nil
}

// LoadV1 loads the data of this `OrgUserInvitation` instance
// given the `.Id` property. If the `.Id` property is the
// zero-value, this function returns an error. If no error
// is returned, this `OrgUserInvitation` instance can be
// expected to be populated with data from the database
func (oui *OrgUserInvitation) LoadV1(opts DatabaseConnection) error {
	if err := oui.assertIdDefined(); err != nil {
		return err
	}
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT
				inviter_id,
				acceptor_id,
				acceptor_email,
				org_id,
				join_code,
				type,
				created_at,
				last_updated_at
				FROM org_user_invitations oui
				WHERE id = ?
		`,
		Args:     []any{oui.Id},
		FnSource: "models.OrgUserInvitation.LoadV1",
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(
				&oui.InviterId,
				&oui.AcceptorId,
				&oui.AcceptorEmail,
				&oui.OrgId,
				&oui.JoinCode,
				&oui.Type,
				&oui.CreatedAt,
				&oui.LastUpdatedAt,
			)
		},
	}); err != nil {
		return err
	}
	return nil
}

func (oui *OrgUserInvitation) ReplaceAcceptorEmailWithId(opts DatabaseConnection) error {
	if err := oui.assertAcceptorIdDefined(); err != nil {
		return err
	} else if err := oui.assertIdDefined(); err != nil {
		return err
	}
	if err := executeMysqlUpdate(mysqlQueryInput{
		Db:           opts.Db,
		Stmt:         `UPDATE org_user_invitations SET acceptor_id = ? WHERE id = ?`,
		Args:         []any{*oui.AcceptorId, oui.Id},
		RowsAffected: oneRowAffected,
		FnSource:     "models.OrgUserInvitation.ReplaceAcceptorEmailWithId",
	}); err != nil {
		return err
	}
	return nil
}
