package models

import (
	"database/sql"
	"fmt"
)

type VerifyUserV1Opts struct {
	Db *sql.DB

	VerificationCode string
}

func VerifyUserV1(opts VerifyUserV1Opts) (*User, error) {
	sqlStmt := `
	SELECT
    id,
    email,
    email_verification_code,
		type,
		created_at,
		is_deleted,
		deleted_at,
		is_disabled,
		disabled_at
		FROM users
			WHERE email_verification_code = ?
	`
	sqlArgs := []any{opts.VerificationCode}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("models.VerifyUserV1: failed to prepare insert statement: %s", err)
	}

	row := stmt.QueryRow(sqlArgs...)
	if err != nil {
		return nil, fmt.Errorf("models.VerifyUserV1: failed to query statement: %s", err)
	}
	userInstance := User{}
	if err := row.Scan(
		&userInstance.Id,
		&userInstance.Email,
		&userInstance.EmailVerificationCode,
		&userInstance.Type,
		&userInstance.CreatedAt,
		&userInstance.IsDeleted,
		&userInstance.DeletedAt,
		&userInstance.IsDisabled,
		&userInstance.DisabledAt,
	); err != nil {
		return nil, fmt.Errorf("failed to get user row: %s", err)
	}

	sqlStmt = `
	UPDATE users
		SET email_verification_code = ''
		WHERE id = ?;
	`
	sqlArgs = []any{*userInstance.Id}
	stmt, err = opts.Db.Prepare(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("models.VerifyUserV1: failed to prepare insert statement: %s", err)
	}

	_, err = stmt.Exec(sqlArgs...)
	if err != nil {
		return nil, fmt.Errorf("models.VerifyUserV1: failed to query statement: %s", err)
	}

	return &userInstance, nil
}
