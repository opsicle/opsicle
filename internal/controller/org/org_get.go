package org

import (
	"database/sql"
	"errors"
	"fmt"
)

type GetV1Opts struct {
	Db *sql.DB

	Id   *string
	Code *string
}

// GetV1 returns an organisation given either it's ID or code;
// when no organisation is found, returns nil for both return values
func GetV1(opts GetV1Opts) (*Org, error) {
	if opts.Id == nil && opts.Code == nil {
		return nil, fmt.Errorf("org.GetV1: either the org id or its code has to be specified")
	}
	selectorField := "id"
	selectorValue := ""
	if opts.Id != nil {
		selectorValue = *opts.Id
	} else if opts.Code != nil {
		selectorField = "code"
		selectorValue = *opts.Code
		fmt.Println("code selected: ", selectorValue)
	}
	stmt, err := opts.Db.Prepare(fmt.Sprintf(`
	SELECT 
		id,
		name,
		created_at,
		updated_at,
		is_deleted,
		deleted_at,
		is_disabled,
		disabled_at,
		code,
		type
		FROM orgs
		WHERE %s = ?`, selectorField))
	if err != nil {
		return nil, fmt.Errorf("org.GetV1: failed to prepare insert statement: %s", err)
	}

	res := stmt.QueryRow(selectorValue)
	if res.Err() != nil {
		return nil, fmt.Errorf("org.GetV1: failed to query org using : %s", err)
	}
	var org Org
	if err := res.Scan(
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
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("org.GetV1: failed to execute insert statement: %s", err)
	}
	return &org, nil
}
