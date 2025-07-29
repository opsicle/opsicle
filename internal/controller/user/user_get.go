package user

import (
	"database/sql"
	"fmt"
	"opsicle/internal/controller/org"
)

type GetV1Opts struct {
	Db *sql.DB

	OrgCode *string
	Id      *string
	Email   *string
}

func GetV1(opts GetV1Opts) (*User, error) {
	selectionField := "`users`.`email`"
	selectionValue := ""
	if opts.Id != nil {
		selectionField = "`users`.`id"
		selectionValue = *opts.Id
	} else if opts.Email != nil {
		selectionValue = *opts.Email
	} else {
		return nil, fmt.Errorf("failed to receive either the user id or email in user.GetV1")
	}
	sqlStmt := fmt.Sprintf(`
	SELECT
    users.id AS user_id,
    users.email,
    users.password_hash,
    orgs.id AS org_id,
    orgs.name AS org_name,
    orgs.code AS org_code
		FROM users
			JOIN org_users ON users.id = org_users.user_id
			JOIN orgs ON org_users.org_id = orgs.id
		WHERE %s = ?`,
		selectionField,
	)
	sqlArgs := []any{selectionValue}
	if opts.OrgCode != nil {
		sqlStmt += " AND orgs.id = ?"
		sqlArgs = append(sqlArgs, *opts.OrgCode)
	}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insert statement: %s", err)
	}

	row := stmt.QueryRow(sqlArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query statement: %s", err)
	}
	userInstance := User{
		Org: &org.Org{},
	}
	if err := row.Scan(
		&userInstance.Id,
		&userInstance.Email,
		&userInstance.PasswordHash,
		&userInstance.Org.Id,
		&userInstance.Org.Name,
		&userInstance.Org.Code,
	); err != nil {
		return nil, fmt.Errorf("failed to get user row: %s", err)
	}
	return &userInstance, nil
}
