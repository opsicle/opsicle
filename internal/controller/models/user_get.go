package models

import (
	"database/sql"
	"errors"
	"fmt"
)

type GetUserV1Opts struct {
	Db *sql.DB

	Id    *string
	Email *string
}

func GetUserV1(opts GetUserV1Opts) (*User, error) {
	selectionField := "`users`.`email`"
	selectionValue := ""
	if opts.Id != nil {
		selectionField = "`users`.`id`"
		selectionValue = *opts.Id
	} else if opts.Email != nil {
		selectionValue = *opts.Email
	} else {
		return nil, fmt.Errorf("failed to receive either the user id or email in models.GetUserV1")
	}
	sqlStmt := fmt.Sprintf(`
	SELECT
    users.id AS user_id,
    users.email,
    users.is_email_verified,
    users.email_verification_code,
    users.email_verified_at,
    users.password_hash,
    users.is_disabled,
    users.disabled_at,
    users.is_deleted,
    users.deleted_at
		FROM users
			WHERE %s = ?`,
		selectionField,
	)
	sqlArgs := []any{selectionValue}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insert statement: %s", err)
	}

	row := stmt.QueryRow(sqlArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query statement: %s", err)
	}
	userInstance := User{}
	if err := row.Scan(
		&userInstance.Id,
		&userInstance.Email,
		&userInstance.IsEmailVerified,
		&userInstance.EmailVerificationCode,
		&userInstance.EmailVerifiedAt,
		&userInstance.PasswordHash,
		&userInstance.IsDisabled,
		&userInstance.DisabledAt,
		&userInstance.IsDeleted,
		&userInstance.DeletedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to get a user: %w", ErrorNotFound)
		}
		return nil, fmt.Errorf("failed to query database: %s", err)
	}
	return &userInstance, nil
}
