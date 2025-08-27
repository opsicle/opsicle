package models

import (
	"database/sql"
	"fmt"
	"strings"
)

func atMostNRowsAffected(expected int64) func(int64) bool {
	return func(observed int64) bool {
		return observed <= expected
	}
}

func atLeastNRowsAffected(expected int64) func(int64) bool {
	return func(observed int64) bool {
		return observed >= expected
	}
}

func nRowsAffected(expected int64) func(int64) bool {
	return func(observed int64) bool {
		return observed == expected
	}
}

func oneRowAffected(observed int64) bool {
	return observed == 1
}

type mysqlQueryInput struct {
	Db           *sql.DB
	Stmt         string
	Args         []any
	RowsAffected func(int64) bool
	FnSource     string
}

func executeMysqlDelete(opts mysqlQueryInput) error {
	inputStmt := strings.TrimSpace(opts.Stmt)
	inputOp := strings.SplitN(inputStmt, " ", 2)
	if strings.ToLower(inputOp[0]) != "delete" {
		return fmt.Errorf("")
	}
	stmt, err := opts.Db.Prepare(inputStmt)
	if err != nil {
		return fmt.Errorf("%s: failed to prepare delete statement: %w", opts.FnSource, ErrorStmtPreparationFailed)
	}
	results, err := stmt.Exec(opts.Args...)
	if err != nil {
		return fmt.Errorf("%s: failed to execute delete statement: %w", opts.FnSource, ErrorQueryFailed)
	}
	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: failed to get n(rows) deleted: %w", opts.FnSource, ErrorRowsAffectedCheckFailed)
	}
	if !opts.RowsAffected(rowsAffected) {
		return fmt.Errorf("%s: n(rows) deleted was wrong (got %v): %w", opts.FnSource, rowsAffected, ErrorRowsAffectedCheckFailed)
	}
	return nil
}

func executeMysqlUpdate(opts mysqlQueryInput) error {
	inputStmt := strings.TrimSpace(opts.Stmt)
	inputOp := strings.SplitN(inputStmt, " ", 2)
	if strings.ToLower(inputOp[0]) != "update" {
		return fmt.Errorf("")
	}
	stmt, err := opts.Db.Prepare(inputStmt)
	if err != nil {
		return fmt.Errorf("%s: failed to prepare update statement: %w", opts.FnSource, ErrorStmtPreparationFailed)
	}
	results, err := stmt.Exec(opts.Args...)
	if err != nil {
		return fmt.Errorf("%s: failed to execute update statement: %w", opts.FnSource, ErrorQueryFailed)
	}
	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: failed to get n(rows) updated: %w", opts.FnSource, ErrorRowsAffectedCheckFailed)
	}
	if !opts.RowsAffected(rowsAffected) {
		return fmt.Errorf("%s: n(rows) updated was wrong (got %v): %w", opts.FnSource, rowsAffected, ErrorRowsAffectedCheckFailed)
	}
	return nil
}
