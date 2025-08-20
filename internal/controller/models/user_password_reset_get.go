package models

import (
	"database/sql"
	"errors"
	"fmt"
)

type GetUserPasswordResetV1Input struct {
	Db *sql.DB

	VerificationCode string
}

type GetUserPasswordResetV1Output struct {
	Id     string
	UserId string
}

func GetUserPasswordResetV1(opts GetUserPasswordResetV1Input) (*GetUserPasswordResetV1Output, error) {
	sqlStmt := `
	SELECT 
		id,
		user_id
	FROM user_password_reset
	WHERE
		verification_code = ?
		AND status = ?`
	sqlArgs := []any{
		opts.VerificationCode,
		"pending",
	}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("models.GetUserPasswordResetV1: failed to prepare insert statement: %w", err)
	}
	row := stmt.QueryRow(sqlArgs...)
	if row.Err() != nil {
		return nil, fmt.Errorf("models.GetUserPasswordResetV1: failed to execute statement: %w", row.Err())
	}
	output := GetUserPasswordResetV1Output{}
	if err := row.Scan(
		&output.Id,
		&output.UserId,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("models.GetUserPasswordResetV1: failed to get a password reset attempt: %w", ErrorNotFound)
		}
		return nil, fmt.Errorf("models.GetUserPasswordResetV1: failed to get a password reset attempt: %w", err)
	}
	return &output, nil
}
