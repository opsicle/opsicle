package models

import (
	"database/sql"
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
	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
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
				?,
				?
			)
		`,
		Args: []any{
			userLoginId,
			opts.UserId,
			opts.IpAddress,
			opts.UserAgent,
			opts.RequiresMfa,
			time.Now().Add(5 * time.Minute),
			"pending",
		},
	}); err != nil {
		return "", err
	}
	return userLoginId, nil
}
