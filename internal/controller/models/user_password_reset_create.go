package models

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type CreateUserPasswordResetV1Input struct {
	Db *sql.DB

	UserId           string
	IpAddress        string
	UserAgent        string
	VerificationCode string
}

func CreateUserPasswordResetV1(opts CreateUserPasswordResetV1Input) (string, error) {
	passwordResetId := uuid.NewString()
	sqlStmt := `
	INSERT INTO user_password_reset(
		id,
		user_id,
		ip_address,
		user_agent,
		verification_code,
		expires_at,
		status
	) VALUES (
		?,
		?,
		?,
		?,
		?,
		?,
		?
	)`
	sqlArgs := []any{
		passwordResetId,
		opts.UserId,
		opts.IpAddress,
		opts.UserAgent,
		opts.VerificationCode,
		time.Now().Add(5 * time.Minute),
		"pending",
	}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return "", fmt.Errorf("models.CreateUserPasswordResetV1: failed to prepare insert statement: %w", err)
	}
	if _, err := stmt.Exec(sqlArgs...); err != nil {
		return "", fmt.Errorf("models.CreateUserPasswordResetV1: failed to execute statement: %w", err)
	}
	return passwordResetId, nil
}
