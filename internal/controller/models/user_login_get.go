package models

import (
	"database/sql"
)

type GetUserLoginV1Input struct {
	Db *sql.DB

	LoginId string
}

func GetUserLoginV1(opts GetUserLoginV1Input) (*UserLogin, error) {
	userLogin := UserLogin{}
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT 
				id,
				user_id,
				ip_address,
				user_agent,
				is_pending_mfa,
				expires_at
				FROM user_login
					WHERE id = ?
			`,
		Args:     []any{opts.LoginId},
		FnSource: "models.GetUserLoginV1",
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(
				&userLogin.Id,
				&userLogin.UserId,
				&userLogin.IpAddress,
				&userLogin.UserAgent,
				&userLogin.IsPendingMfa,
				&userLogin.ExpiresAt,
			)
		},
	}); err != nil {
		return nil, err
	}
	return &userLogin, nil
}
