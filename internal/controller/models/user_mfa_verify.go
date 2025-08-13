package models

import (
	"database/sql"
	"fmt"
)

type VerifyUserMfaV1Opts struct {
	Db *sql.DB

	Id string
}

func VerifyUserMfaV1(opts VerifyUserMfaV1Opts) error {
	sqlArgs := []any{opts.Id}
	sqlStmt := `
	UPDATE user_mfa SET
		is_verified = true,
		verified_at = now()
		WHERE id = ?
	`
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return fmt.Errorf("models.VerifyUserMfaV1: failed to prepare insert statement: %s", err)
	}

	results, err := stmt.Exec(sqlArgs...)
	if err != nil {
		return fmt.Errorf("models.VerifyUserMfaV1: failed to execute query: %s", err)
	}
	if rowsAffected, err := results.RowsAffected(); err != nil {
		return fmt.Errorf("models.VerifyUserMfaV1: failed to get created row: %s", err)
	} else if rowsAffected == 0 {
		return fmt.Errorf("models.VerifyUserMfaV1: failed to create a row: %s", err)
	}

	return nil
}
