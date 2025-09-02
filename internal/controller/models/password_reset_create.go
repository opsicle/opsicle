package models

import (
	"database/sql"
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
	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
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
			)`,
		Args: []any{
			passwordResetId,
			opts.UserId,
			opts.IpAddress,
			opts.UserAgent,
			opts.VerificationCode,
			time.Now().Add(5 * time.Minute),
			"pending",
		},
		FnSource:     "models.CreateUserPasswordResetV1",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return "", nil
	}

	return passwordResetId, nil
}
