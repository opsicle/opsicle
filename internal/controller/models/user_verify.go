package models

import (
	"database/sql"
	"errors"
	"fmt"
)

type VerifyUserV1Opts struct {
	Db *sql.DB

	VerificationCode string
	UserAgent        string
	IpAddress        string
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
		return nil, fmt.Errorf("models.VerifyUserV1: failed to prepare insert statement: %w", err)
	}

	row := stmt.QueryRow(sqlArgs...)
	if err != nil {
		return nil, fmt.Errorf("models.VerifyUserV1: failed to query statement: %w", err)
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no rows found: %w", ErrorNotFound)
		}
		return nil, fmt.Errorf("failed to get user row: %w", err)
	}

	sqlStmt = `
	UPDATE users
		SET email_verification_code = '',
				is_email_verified = true,
				email_verified_at = NOW(),
				email_verified_by_user_agent = ?,
				email_verified_by_ip_address = ?
		WHERE id = ?;
	`
	sqlArgs = []any{
		opts.UserAgent,
		opts.IpAddress,
		*userInstance.Id,
	}
	stmt, err = opts.Db.Prepare(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("models.VerifyUserV1: failed to prepare insert statement: %w", err)
	}

	_, err = stmt.Exec(sqlArgs...)
	if err != nil {
		return nil, fmt.Errorf("models.VerifyUserV1: failed to query statement: %w", err)
	}

	return &userInstance, nil
}
