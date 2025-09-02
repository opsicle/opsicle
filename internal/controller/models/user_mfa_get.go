package models

import (
	"database/sql"
)

type GetUserMfaV1Opts struct {
	Db *sql.DB

	Id string
}

func GetUserMfaV1(opts GetUserMfaV1Opts) (*UserMfa, error) {
	userMfa := UserMfa{}
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT
				id,
				type,
				secret,
				config_json,
				is_verified,
				verified_at,
				created_at,
				last_updated_at
				FROM user_mfa
					WHERE id = ?
		`,
		Args:     []any{opts.Id},
		FnSource: "models.GetUserMfaV1",
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(
				&userMfa.Id,
				&userMfa.Type,
				&userMfa.Secret,
				&userMfa.ConfigJson,
				&userMfa.IsVerified,
				&userMfa.VerifiedAt,
				&userMfa.CreatedAt,
				&userMfa.LastUpdatedAt,
			)
		},
	}); err != nil {
		return nil, err
	}

	return &userMfa, nil
}
