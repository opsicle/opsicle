package org

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type CreateV1Opts struct {
	Db *sql.DB

	Name string
	Code string
	Type Type
}

func CreateV1(opts CreateV1Opts) error {
	orgUuid := uuid.NewString()
	stmt, err := opts.Db.Prepare(`
	INSERT INTO orgs(
		id,
		name,
		code,
		type
	) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %s", err)
	}

	res, err := stmt.Exec(orgUuid, opts.Name, opts.Code, string(opts.Type))
	if err != nil {
		return fmt.Errorf("failed to execute insert statement: %s", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve the number of rows affected: %s", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("failed to insert only 1 user")
	}
	return nil
}
