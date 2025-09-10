package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
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
	ProcessRows  func(*sql.Rows) error
	ProcessRow   func(*sql.Row) error
}

func executeMysqlDelete(opts mysqlQueryInput) error {
	if opts.Db == nil {
		return fmt.Errorf("%s: missing db input: %w", opts.FnSource, ErrorDatabaseUndefined)
	}
	inputStmt := strings.TrimSpace(opts.Stmt)
	inputOp := strings.SplitN(strings.ReplaceAll(inputStmt, "\n", " "), " ", 2)
	if strings.ToLower(inputOp[0]) != "delete" {
		return fmt.Errorf("only 'delete' statements are allowed")
	}
	stmt, err := opts.Db.Prepare(inputStmt)
	if err != nil {
		return fmt.Errorf("%s: failed to prepare delete statement: %w (%w)", opts.FnSource, ErrorStmtPreparationFailed, err)
	}
	results, err := stmt.Exec(opts.Args...)
	if err != nil {
		return fmt.Errorf("%s: failed to execute delete statement: %w (%w)", opts.FnSource, ErrorDeleteFailed, err)
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

func executeMysqlInsert(opts mysqlQueryInput) error {
	if opts.Db == nil {
		return fmt.Errorf("%s: missing db input: %w", opts.FnSource, ErrorDatabaseUndefined)
	}
	inputStmt := strings.TrimSpace(opts.Stmt)
	inputOp := strings.SplitN(strings.ReplaceAll(inputStmt, "\n", " "), " ", 2)
	if strings.ToLower(inputOp[0]) != "insert" {
		return fmt.Errorf("%s: only 'insert' statements are allowed: %w", opts.FnSource, ErrorInvalidInput)
	}
	stmt, err := opts.Db.Prepare(inputStmt)
	if err != nil {
		return fmt.Errorf("%s: failed to prepare insert statement: %w (%w)", opts.FnSource, ErrorStmtPreparationFailed, err)
	}
	results, err := stmt.Exec(opts.Args...)
	if err != nil {
		if isMysqlDuplicateError(err) {
			return fmt.Errorf("%s: duplicate detected: %w: %w", opts.FnSource, ErrorDuplicateEntry, err)
		}
		return fmt.Errorf("%s: failed to execute insert statement: %w (%w)", opts.FnSource, ErrorInsertFailed, err)
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

func executeMysqlSelect(opts mysqlQueryInput) error {
	if opts.Db == nil {
		return fmt.Errorf("%s: missing db input: %w", opts.FnSource, ErrorDatabaseUndefined)
	}
	inputStmt := strings.TrimSpace(opts.Stmt)
	inputOp := strings.SplitN(strings.ReplaceAll(inputStmt, "\n", " "), " ", 2)
	if strings.ToLower(inputOp[0]) != "select" {
		return fmt.Errorf("%s: only 'select' statements are allowed: %w", opts.FnSource, ErrorInvalidInput)
	}
	if opts.ProcessRow == nil {
		return fmt.Errorf("%s: ProcessRow is undefined: %w", opts.FnSource, ErrorInvalidInput)
	}
	stmt, err := opts.Db.Prepare(inputStmt)
	if err != nil {
		return fmt.Errorf("%s: failed to prepare select statement: %w (%w)", opts.FnSource, ErrorStmtPreparationFailed, err)
	}
	row := stmt.QueryRow(opts.Args...)
	if row.Err() != nil {
		return fmt.Errorf("%s: failed to execute select statement: %w (%w)", opts.FnSource, ErrorSelectFailed, row.Err())
	}
	if err := opts.ProcessRow(row); err != nil {
		if isMysqlNotFoundError(err) {
			return fmt.Errorf("%s: no rows: %w: %w", opts.FnSource, ErrorNotFound, err)
		}
		return fmt.Errorf("%s: failed to process result: %w", opts.FnSource, err)
	}
	return nil
}

func executeMysqlSelects(opts mysqlQueryInput) error {
	if opts.Db == nil {
		return fmt.Errorf("%s: missing db input: %w", opts.FnSource, ErrorDatabaseUndefined)
	}
	inputStmt := strings.TrimSpace(opts.Stmt)
	inputOp := strings.SplitN(strings.ReplaceAll(inputStmt, "\n", " "), " ", 2)
	if strings.ToLower(inputOp[0]) != "select" {
		return fmt.Errorf("%s: only 'select' statements are allowed: %w", opts.FnSource, ErrorInvalidInput)
	}
	if opts.ProcessRows == nil {
		return fmt.Errorf("%s: ProcessRows is undefined: %w", opts.FnSource, ErrorInvalidInput)
	}
	stmt, err := opts.Db.Prepare(inputStmt)
	if err != nil {
		return fmt.Errorf("%s: failed to prepare select statement: %w (%w)", opts.FnSource, ErrorStmtPreparationFailed, err)
	}
	rows, err := stmt.Query(opts.Args...)
	if err != nil {
		return fmt.Errorf("%s: failed to execute select statement: %w (%w)", opts.FnSource, ErrorSelectsFailed, err)
	}
	counter := 0
	for rows.Next() {
		if err := opts.ProcessRows(rows); err != nil {
			if isMysqlNotFoundError(err) {
				return fmt.Errorf("%s: no rows: %w", opts.FnSource, ErrorNotFound)
			}
			return fmt.Errorf("%s: failed to process row[%v]: %w", opts.FnSource, counter, err)
		}
		counter++
	}
	return nil
}

func executeMysqlUpdate(opts mysqlQueryInput) error {
	inputStmt := strings.TrimSpace(opts.Stmt)
	inputOp := strings.SplitN(strings.ReplaceAll(inputStmt, "\n", " "), " ", 2)
	if strings.ToLower(inputOp[0]) != "update" {
		return fmt.Errorf("%s: only 'update' statements are allowed: %w", opts.FnSource, ErrorInvalidInput)
	}
	stmt, err := opts.Db.Prepare(inputStmt)
	if err != nil {
		return fmt.Errorf("%s: failed to prepare update statement: %w (%w)", opts.FnSource, ErrorStmtPreparationFailed, err)
	}
	results, err := stmt.Exec(opts.Args...)
	if err != nil {
		return fmt.Errorf("%s: failed to execute update statement: %w (%w)", opts.FnSource, ErrorUpdateFailed, err)
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

func isMysqlNotFoundError(err error) bool {
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	return false
}

func isMysqlDuplicateError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		if mysqlErr.Number == mysqlErrorDuplicateEntryCode {
			return true
		}
	}
	return false
}
