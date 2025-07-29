package user

import (
	"database/sql"
	"fmt"
	"opsicle/internal/controller/org"
)

type ListV1Opts struct {
	Db *sql.DB

	OrgCode string
}

func ListV1(opts ListV1Opts) ([]User, error) {
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
		return nil, fmt.Errorf("failed to prepare insert statement: %s", err)
	}

	rows, err := stmt.Query(sqlArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query statement: %s", err)
	}
	userInstances := []User{}
	for rows.Next() {
		userInstance := User{Org: &org.Org{}}
		if err := rows.Scan(
			&userInstance.Org.Id,
			&userInstance.Org.Name,
			&userInstance.Org.Code,
			&userInstance.Id,
			&userInstance.Email,
		); err != nil {
			return nil, fmt.Errorf("failed to get user row: %s", err)
		}
		userInstances = append(userInstances, userInstance)
	}

	return userInstances, nil
}
