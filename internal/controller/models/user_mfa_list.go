package models

import (
	"database/sql"
	"errors"
	"fmt"
)

type ListUserMfasV1Opts struct {
	Db *sql.DB

	UserId string
}

func ListUserMfasV1(opts ListUserMfasV1Opts) ([]UserMfa, error) {
	sqlStmt := `
	SELECT
		id,
    type,
		is_verified,
		verified_at,
		created_at,
		last_updated_at
		FROM user_mfa
			WHERE user_id = ?
				AND is_verified = true
	`
	sqlArgs := []any{opts.UserId}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("models.ListUserMfasV1: failed to prepare insert statement: %s", err)
	}

	rows, err := stmt.Query(sqlArgs...)
	if err != nil {
		return nil, fmt.Errorf("models.ListUserMfasV1: failed to query: %s", err)
	}

	output := []UserMfa{}
	for rows.Next() {
		userMfa := UserMfa{}
		if err := rows.Scan(
			&userMfa.Id,
			&userMfa.Type,
			&userMfa.IsVerified,
			&userMfa.VerifiedAt,
			&userMfa.CreatedAt,
			&userMfa.LastUpdatedAt,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, fmt.Errorf("models.ListUserMfasV1: failed to get a user_mfa row: %s", err)
		}
		output = append(output, userMfa)
	}
	return output, nil
}
