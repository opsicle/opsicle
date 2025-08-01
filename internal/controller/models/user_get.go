package models

import (
	"database/sql"
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
		selectionField = "`users`.`id"
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
    users.email_verification_code,
    users.password_hash AS password_hash
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
		&userInstance.EmailVerificationCode,
		&userInstance.PasswordHash,
	); err != nil {
		return nil, fmt.Errorf("failed to get user row: %s", err)
	}
	return &userInstance, nil
}
