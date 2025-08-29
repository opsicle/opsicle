package models

import (
	"database/sql"
	"errors"
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
	Type          string     `json:"type" yaml:"type"`
	CreatedAt     time.Time  `json:"createdAt" yaml:"createdAt"`
	LastUpdatedAt *time.Time `json:"lastUpdatedAt" yaml:"lastUpdatedAt"`
}

func (oui *OrgUserInvitation) DeleteById(opts DatabaseConnection) error {
	sqlStmt := "DELETE FROM org_user_invitations WHERE id = ?"
	sqlArgs := []any{oui.Id}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return fmt.Errorf("models.OrgUserInvitation.DeleteById: failed to prepare insert statement: %w", err)
	}
	results, err := stmt.Exec(sqlArgs...)
	if err != nil {
		return fmt.Errorf("models.OrgUserInvitation.DeleteById: failed to prepare insert statement: %w", err)
	}
	if nRows, err := results.RowsAffected(); err != nil {
		return fmt.Errorf("models.OrgUserInvitation.DeleteById: failed to verify execution effects: %w", err)
	} else if nRows == 0 {
		return fmt.Errorf("models.OrgUserInvitation.DeleteById: failed to verify execution outcome: %w", err)
	}
	return nil
}

// LoadV1 loads the data of this `OrgUserInvitation` instance
// given the `.Id` property. If the `.Id` property is the
// zero-value, this function returns an error. If no error
// is returned, this `OrgUserInvitation` instance can be
// expected to be populated with data from the database
func (oui *OrgUserInvitation) LoadV1(opts DatabaseConnection) error {
	if oui.Id == "" {
		return fmt.Errorf("invitation id invalid: %w", ErrorInvalidInput)
	}
	sqlStmt := `
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
	`
	sqlArgs := []any{oui.Id}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return fmt.Errorf("models.OrgUserInvitation.LoadV1: failed to prepare insert statement: %w", err)
	}
	row := stmt.QueryRow(sqlArgs...)
	if row.Err() != nil {
		return fmt.Errorf("models.OrgUserInvitation.LoadV1: failed to execute statement: %w", err)
	}
	if err := row.Scan(
		&oui.InviterId,
		&oui.AcceptorId,
		&oui.AcceptorEmail,
		&oui.OrgId,
		&oui.JoinCode,
		&oui.Type,
		&oui.CreatedAt,
		&oui.LastUpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("models.OrgUserInvitation.LoadV1: failed to retrieve invitation: %w: %w", ErrorNotFound, err)
		}
		return fmt.Errorf("models.OrgUserInvitation.LoadV1: failed to retrieve invitation: %w", err)
	}

	return nil
}

func (oui *OrgUserInvitation) ReplaceAcceptorEmailWithId(opts DatabaseConnection) error {
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
