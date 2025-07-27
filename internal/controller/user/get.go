package user

import (
	"database/sql"
	"fmt"
)

type GetV1Opts struct {
	Db *sql.DB

	Id    *string
	Email *string
}

func GetV1(opts GetV1Opts) (*User, error) {
	selectionField := "email"
	selectionValue := ""
	if opts.Id != nil {
		selectionField = "id"
		selectionValue = *opts.Id
	} else if opts.Email != nil {
		selectionValue = *opts.Email
	} else {
		return nil, fmt.Errorf("failed to receive either the user id or email in user.GetV1")
	}
	stmt, err := opts.Db.Prepare(fmt.Sprintf(`
	SELECT 
		id,
		email,
		password_hash
		FROM users
		WHERE %s = ?`, selectionField))
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insert statement: %s", err)
	}

	row := stmt.QueryRow(selectionValue)
	if err != nil {
		return nil, fmt.Errorf("failed to query statement: %s", err)
	}
	var userInstance User
	if err := row.Scan(
		&userInstance.Id,
		&userInstance.Email,
		&userInstance.PasswordHash,
	); err != nil {
		return nil, fmt.Errorf("failed to get user row: %s", err)
	}
	return &userInstance, nil
}
