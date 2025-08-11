package models

import (
	"database/sql"
	"errors"
	"fmt"
)

type ListUserOrgsV1Opts struct {
	Db *sql.DB

	UserId string
}

// ListUserOrgsV1 returns an organisation given either it's ID or code;
// when no organisation is found, returns nil for both return values
func ListUserOrgsV1(opts ListUserOrgsV1Opts) ([]Org, error) {
	sqlStmt := `
	SELECT 
		o.id,
		o.name,
		o.created_at,
		o.last_updated_at,
		o.is_deleted,
		o.deleted_at,
		o.is_disabled,
		o.disabled_at,
		o.code,
		o.type as org_type,
		ou.type as member_type,
		ou.joined_at
		FROM orgs o
			JOIN org_users ou ON o.id = ou.org_id
		WHERE ou.user_id = ?
	`
	sqlArgs := []any{opts.UserId}

	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("models.ListUserOrgsV1: failed to prepare insert statement: %s", err)
	}

	rows, err := stmt.Query(sqlArgs...)
	if err != nil {
		return nil, fmt.Errorf("models.ListUserOrgsV1: failed to query org using : %s", err)
	}
	output := []Org{}
	for rows.Next() {
		var org Org
		if err := rows.Scan(
			&org.Id,
			&org.Name,
			&org.CreatedAt,
			&org.UpdatedAt,
			&org.IsDeleted,
			&org.DeletedAt,
			&org.IsDisabled,
			&org.DisabledAt,
			&org.Code,
			&org.Type,
			&org.MemberType,
			&org.JoinedAt,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, fmt.Errorf("models.ListUserOrgsV1: failed to scan row into Org struct: %s", err)
		}
		output = append(output, org)
	}
	return output, nil
}
