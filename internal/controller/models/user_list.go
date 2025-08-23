package models

import (
	"database/sql"
	"fmt"
)

type ListUsersV1Opts struct {
	Db *sql.DB

	OrgCode string
}

func ListUsersV1(opts ListUsersV1Opts) ([]User, error) {
	sqlStmt := `
	SELECT
		orgs.id AS org_id,
    orgs.name AS org_name,
    orgs.code AS org_code,
    users.id AS user_id,
    users.email AS user_email
		FROM orgs
			JOIN org_users ON orgs.id = org_users.org_id
			JOIN users ON org_users.user_id = users.id
		WHERE orgs.code = ?`
	sqlArgs := []any{opts.OrgCode}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insert statement: %w", err)
	}

	rows, err := stmt.Query(sqlArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query statement: %w", err)
	}
	userInstances := []User{}
	for rows.Next() {
		userInstance := User{Org: &Org{}}
		if err := rows.Scan(
			&userInstance.Org.Id,
			&userInstance.Org.Name,
			&userInstance.Org.Code,
			&userInstance.Id,
			&userInstance.Email,
		); err != nil {
			return nil, fmt.Errorf("failed to get user row: %w", err)
		}
		userInstances = append(userInstances, userInstance)
	}

	return userInstances, nil
}
