package models

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type CreateUserLoginV1Input struct {
	Db *sql.DB

	UserId      string
	IpAddress   string
	UserAgent   string
	RequiresMfa bool
	Status      string
}

func CreateUserLoginV1(opts CreateUserLoginV1Input) (string, error) {
	userLoginId := uuid.NewString()
	sqlStmt := `
	INSERT INTO user_login(
		id,
		user_id,
		ip_address,
		user_agent,
		is_pending_mfa,
		expires_at,
		status
	) VALUES (
		?,
		?,
		?,
		?,
		?,
		?
	)`
	sqlArgs := []any{
		userLoginId,
		opts.UserId,
		opts.IpAddress,
		opts.UserAgent,
		opts.RequiresMfa,
		time.Now().Add(5 * time.Minute),
		"pending",
	}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return "", fmt.Errorf("models.CreateUserLoginV1: failed to prepare insert statement: %w", err)
	}
	if _, err := stmt.Exec(sqlArgs...); err != nil {
		return "", fmt.Errorf("models.CreateUserLoginV1: failed to execute statement: %w", err)
	}
	return userLoginId, nil
}
