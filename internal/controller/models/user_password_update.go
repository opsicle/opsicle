package models

import (
	"database/sql"
	"fmt"
	"opsicle/internal/auth"
)

type UpdateUserPasswordV1Input struct {
	Db *sql.DB

	UserId      string
	NewPassword string
}

func UpdateUserPasswordV1(opts UpdateUserPasswordV1Input) error {
	passwordHash, err := auth.HashPassword(opts.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	sqlStmt := `UPDATE users SET password_hash = ? WHERE id = ?`
	sqlArgs := []any{passwordHash, opts.UserId}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return fmt.Errorf("models.UpdateUserPasswordV1: failed to prepare insert statement: %w", err)
	}
	results, err := stmt.Exec(sqlArgs...)
	if err != nil {
		return fmt.Errorf("models.UpdateUserPasswordV1: failed to execute statement: %w", err)
	}
	if nRows, err := results.RowsAffected(); err != nil {
		return fmt.Errorf("models.UpdateUserPasswordV1: failed to verify execution effects: %w", err)
	} else if nRows == 0 {
		return fmt.Errorf("models.UpdateUserPasswordV1: failed to verify execution outcome: %w", err)
	}
	return nil
}
