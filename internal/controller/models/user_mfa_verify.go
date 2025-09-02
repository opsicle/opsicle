package models

import (
	"database/sql"
)

type VerifyUserMfaV1Opts struct {
	Db *sql.DB

	Id string
}

func VerifyUserMfaV1(opts VerifyUserMfaV1Opts) error {
	return executeMysqlUpdate(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			UPDATE user_mfa SET
				is_verified = true,
				verified_at = now()
				WHERE id = ?
		`,
		Args:         []any{opts.Id},
		FnSource:     "models.VerifyUserMfaV1",
		RowsAffected: oneRowAffected,
	})
}
