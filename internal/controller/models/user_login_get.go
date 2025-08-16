package models

import (
	"database/sql"
	"fmt"
)

type GetUserLoginV1Input struct {
	Db *sql.DB

	LoginId string
}

func GetUserLoginV1(opts GetUserLoginV1Input) (*UserLogin, error) {
	sqlStmt := `
	SELECT 
		id,
		user_id,
		ip_address,
		user_agent,
		is_pending_mfa,
		expires_at
		FROM user_login
			WHERE id = ?
	`
	sqlArgs := []any{opts.LoginId}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("models.GetUserLoginV1: failed to prepare insert statement: %w", err)
	}
	row := stmt.QueryRow(sqlArgs...)
	if row.Err() != nil {
		return nil, fmt.Errorf("models.GetUserLoginV1: failed to execute statement: %w", err)
	}
	userLogin := UserLogin{}
	if err := row.Scan(
		&userLogin.Id,
		&userLogin.UserId,
		&userLogin.IpAddress,
		&userLogin.UserAgent,
		&userLogin.IsPendingMfa,
		&userLogin.ExpiresAt,
	); err != nil {
		return nil, fmt.Errorf("models.GetUserLoginV1: failed to retrieve user login: %w", err)
	}
	return &userLogin, nil
}
