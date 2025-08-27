package models

import (
	"database/sql"
	"errors"
	"fmt"
)

// LoadV1 loads an organisation user based on the `UserId` and `OrgId`,
// if these are empty or not UUIDs, this function will return an
// ErrorInvalidInput error
func (ou *OrgUser) LoadV1(opts DatabaseConnection) error {
	if err := ou.validate(); err != nil {
		return err
	}
	sqlStmt := `
	SELECT 
		ou.joined_at,
		ou.type,
		u.email,
		u.type,
		o.code,
		o.name,
		FROM org_users ou
			JOIN users u ON ou.user_id = u.id
			JOIN orgs o ON ou.org_id = o.id
		WHERE 
			ou.org_id = ?
			AND ou.user_id = ?
	`
	sqlArgs := []any{ou.OrgId, ou.UserId}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return fmt.Errorf("models.OrgUser.LoadV1: failed to prepare statement: %w", ErrorStmtPreparationFailed)
	}

	res := stmt.QueryRow(sqlArgs...)
	if res.Err() != nil {
		return fmt.Errorf("models.OrgUser.LoadV1: failed to execute statement: %w", ErrorQueryFailed)
	}
	if err := res.Scan(
		&ou.JoinedAt,
		&ou.MemberType,
		&ou.UserEmail,
		&ou.UserType,
		&ou.OrgCode,
		&ou.OrgName,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrorNotFound
		}
		return fmt.Errorf("models.OrgUser.LoadV1: failed to load selected data into memory: %w", err)
	}
	return nil
}
