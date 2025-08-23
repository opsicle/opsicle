package models

import (
	"database/sql"
	"errors"
	"fmt"
)

type GetUserMfaV1Opts struct {
	Db *sql.DB

	Id string
}

func GetUserMfaV1(opts GetUserMfaV1Opts) (*UserMfa, error) {
	sqlStmt := `
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
	`
	sqlArgs := []any{opts.Id}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("models.GetUserMfaV1: failed to prepare insert statement: %w", err)
	}

	row := stmt.QueryRow(sqlArgs...)
	if row.Err() != nil {
		return nil, fmt.Errorf("models.GetUserMfaV1: failed to query: %w", err)
	}

	userMfa := UserMfa{}
	if err := row.Scan(
		&userMfa.Id,
		&userMfa.Type,
		&userMfa.Secret,
		&userMfa.ConfigJson,
		&userMfa.IsVerified,
		&userMfa.VerifiedAt,
		&userMfa.CreatedAt,
		&userMfa.LastUpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("models.GetUserMfaV1: failed to get a user_mfa row: %w", err)
	}

	return &userMfa, nil
}
