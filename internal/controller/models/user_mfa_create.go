package models

import (
	"database/sql"
)

type CreateUserMfaV1Opts struct {
	Db *sql.DB

	CurrentPassword string
	Type            string
	UserId          string
}

// func CreateUserMfaV1(opts CreateUserMfaV1Opts) ([]UserMfa, error) {
// 	sqlStmt := `
// 	SELECT
// 		id,
//     type,
// 		created_at,
// 		last_updated_at
// 		FROM user_mfa
// 			WHERE user_id = ?
// 	`
// 	sqlArgs := []any{opts.Id}
// 	stmt, err := opts.Db.Prepare(sqlStmt)
// 	if err != nil {
// 		return nil, fmt.Errorf("models.CreateUserMfaV1: failed to prepare insert statement: %s", err)
// 	}

// 	rows, err := stmt.Query(sqlArgs...)
// 	if err != nil {
// 		return nil, fmt.Errorf("models.CreateUserMfaV1: failed to query: %s", err)
// 	}

// 	output := []UserMfa{}
// 	for rows.Next() {
// 		userMfa := UserMfa{}
// 		if err := rows.Scan(
// 			&userMfa.Id,
// 			&userMfa.Type,
// 			&userMfa.CreatedAt,
// 			&userMfa.LastUpdatedAt,
// 		); err != nil {
// 			if errors.Is(err, sql.ErrNoRows) {
// 				return nil, nil
// 			}
// 			return nil, fmt.Errorf("models.CreateUserMfaV1: failed to get a user_mfa row: %s", err)
// 		}
// 		output = append(output, userMfa)
// 	}
// 	return output, nil
// }
