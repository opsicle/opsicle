package models

import (
	"database/sql"
	"fmt"
	"strings"
)

type updateUserPasswordResetV1Input struct {
	Db *sql.DB

	Id          string
	FieldsToSet map[string]any
}

func updateUserPasswordResetV1(opts updateUserPasswordResetV1Input) error {
	fieldsToSet := []string{}
	for field, value := range opts.FieldsToSet {
		switch v := value.(type) {
		case string:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = \"%s\"", field, v))
		case []byte:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = \"%s\"", field, string(v)))
		default:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = %v", field, v))
		}
	}
	sqlStmt := fmt.Sprintf(`UPDATE user_password_reset SET %s WHERE id = ?`, strings.Join(fieldsToSet, ", "))
	sqlArgs := []any{}

	sqlArgs = append(sqlArgs, opts.Id)
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return fmt.Errorf("models.updateUserLoginV1: failed to prepare insert statement: %w", err)
	}
	if _, err := stmt.Exec(sqlArgs...); err != nil {
		return fmt.Errorf("models.updateUserLoginV1: failed to execute statement: %w", err)
	}
	return nil
}

type SetUserPasswordResetToSuccessV1Input struct {
	Db *sql.DB

	Id string
}

func SetUserPasswordResetToSuccessV1(opts SetUserPasswordResetToSuccessV1Input) error {
	return updateUserPasswordResetV1(updateUserPasswordResetV1Input{
		Db: opts.Db,
		Id: opts.Id,
		FieldsToSet: map[string]any{
			"status": "success",
		},
	})
}
