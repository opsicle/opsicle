package models

import (
	"database/sql"
	"fmt"
	"time"
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
	CreatedAt     time.Time  `json:"createdAt" yaml:"createdAt"`
	LastUpdatedAt *time.Time `json:"lastUpdatedAt" yaml:"lastUpdatedAt"`
}

type ReplaceAcceptorEmailWithIdOpts struct {
	Db *sql.DB
}

func (oui *OrgUserInvitation) ReplaceAcceptorEmailWithId(opts ReplaceAcceptorEmailWithIdOpts) error {
	sqlStmt := `UPDATE org_user_invitations SET acceptor_id = ? WHERE id = ?`
	sqlArgs := []any{oui.AcceptorId, oui.Id}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return fmt.Errorf("models.OrgUserInvitation.ReplaceAcceptorEmailWithId: failed to prepare insert statement: %w", err)
	}
	results, err := stmt.Exec(sqlArgs...)
	if err != nil {
		return fmt.Errorf("models.OrgUserInvitation.ReplaceAcceptorEmailWithId: failed to execute statement: %w", err)
	}
	if nRows, err := results.RowsAffected(); err != nil {
		return fmt.Errorf("models.OrgUserInvitation.ReplaceAcceptorEmailWithId: failed to verify execution effects: %w", err)
	} else if nRows == 0 {
		return fmt.Errorf("models.OrgUserInvitation.ReplaceAcceptorEmailWithId: failed to verify execution outcome: %w", err)
	}
	return nil
}
