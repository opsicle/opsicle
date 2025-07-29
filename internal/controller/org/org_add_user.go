package org

import (
	"database/sql"
	"fmt"
)

type AddUserV1Opts struct {
	Db *sql.DB

	UserId string
}

func (o *Org) AddUserV1(opts AddUserV1Opts) error {
	stmt, err := opts.Db.Prepare(`
	INSERT INTO org_users(
		org_id,
		user_id
	) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("org.Org.AddUserV1: failed to prepare insert statement: %s", err)
	}

	res, err := stmt.Exec(o.Id, opts.UserId)
	if err != nil {
		return fmt.Errorf("org.Org.AddUserV1: failed to execute insert statement: %s", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("org.Org.AddUserV1: failed to retrieve the number of rows affected: %s", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("org.Org.AddUserV1: failed to insert only 1 user")
	}
	return nil
}
