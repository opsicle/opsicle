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
	ProcessRow   func(*sql.Rows) error
}

func executeMysqlDelete(opts mysqlQueryInput) error {
	inputStmt := strings.TrimSpace(opts.Stmt)
	inputOp := strings.SplitN(inputStmt, " ", 2)
	if strings.ToLower(inputOp[0]) != "delete" {
		return fmt.Errorf("only 'delete' statements are allowed")
	}
	stmt, err := opts.Db.Prepare(inputStmt)
	if err != nil {
		return fmt.Errorf("%s: failed to prepare delete statement: %w (%w)", opts.FnSource, ErrorStmtPreparationFailed, err)
	}
	results, err := stmt.Exec(opts.Args...)
	if err != nil {
		return fmt.Errorf("%s: failed to execute delete statement: %w (%w)", opts.FnSource, ErrorQueryFailed, err)
	}
	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: failed to get n(rows) deleted: %w (%w)", opts.FnSource, ErrorRowsAffectedCheckFailed, err)
	}
	if !opts.RowsAffected(rowsAffected) {
		return fmt.Errorf("%s: n(rows) deleted was wrong (got %v): %w", opts.FnSource, rowsAffected, ErrorRowsAffectedCheckFailed)
	}
	return nil
}

func executeMysqlSelects(opts mysqlQueryInput) error {
	inputStmt := strings.TrimSpace(opts.Stmt)
	inputOp := strings.SplitN(strings.ReplaceAll(inputStmt, "\n", " "), " ", 2)
	if strings.ToLower(inputOp[0]) != "select" {
		fmt.Printf("received [%s]\n", inputOp[0])
		return fmt.Errorf("only 'select' statements are allowed")
	}
	if opts.ProcessRow == nil {
		return fmt.Errorf("undefined ProcessRow function")
	}
	stmt, err := opts.Db.Prepare(inputStmt)
	if err != nil {
		return fmt.Errorf("%s: failed to prepare select statement: %w (%w)", opts.FnSource, ErrorStmtPreparationFailed, err)
	}
	rows, err := stmt.Query(opts.Args...)
	if err != nil {
		return fmt.Errorf("%s: failed to execute select statement: %w (%w)", opts.FnSource, ErrorSelectFailed, err)
	}
	counter := 0
	for rows.Next() {
		if err := opts.ProcessRow(rows); err != nil {
			return fmt.Errorf("%s: failed to process row[%v]: %w", opts.FnSource, counter, err)
		}
		counter++
	}
	return nil
}

func executeMysqlUpdate(opts mysqlQueryInput) error {
	inputStmt := strings.TrimSpace(opts.Stmt)
	inputOp := strings.SplitN(inputStmt, " ", 2)
	if strings.ToLower(inputOp[0]) != "update" {
		return fmt.Errorf("only 'update' statements are allowed")
	}
	stmt, err := opts.Db.Prepare(inputStmt)
	if err != nil {
		return fmt.Errorf("%s: failed to prepare update statement: %w (%w)", opts.FnSource, ErrorStmtPreparationFailed, err)
	}
	results, err := stmt.Exec(opts.Args...)
	if err != nil {
		return fmt.Errorf("%s: failed to execute update statement: %w (%w)", opts.FnSource, ErrorQueryFailed, err)
	}
	if opts.RowsAffected != nil {
		rowsAffected, err := results.RowsAffected()
		if err != nil {
			return fmt.Errorf("%s: failed to get n(rows) updated: %w (%w)", opts.FnSource, ErrorRowsAffectedCheckFailed, err)
		}
		if !opts.RowsAffected(rowsAffected) {
			return fmt.Errorf("%s: n(rows) updated was wrong (got %v): %w", opts.FnSource, rowsAffected, ErrorRowsAffectedCheckFailed)
		}
	}
	return nil
}
