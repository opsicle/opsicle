package models

import (
	"database/sql"
	"fmt"
	"strings"
)

type updateUserLoginV1Input struct {
	Db *sql.DB

	Id          string
	FieldsToSet map[string]any
}

func updateUserLoginV1(opts updateUserLoginV1Input) error {
	fieldsToSet := []string{}
	sqlArgs := []any{}
	for field, value := range opts.FieldsToSet {
		switch v := value.(type) {
		case string:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, v)
		case []byte:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, string(v))
		case bool:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, v)
		default:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, fmt.Sprintf("%v", v))
		}
	}
	return executeMysqlUpdate(mysqlQueryInput{
		Db:       opts.Db,
		Stmt:     fmt.Sprintf(`UPDATE user_login SET %s WHERE id = ?`, strings.Join(fieldsToSet, ", ")),
		Args:     append(sqlArgs, opts.Id),
		FnSource: "models.updateUserLoginV1",
	})
}

type SetUserLoginMfaSucceededV1Input struct {
	Db *sql.DB

	Id string
}

func SetUserLoginMfaSucceededV1(opts SetUserLoginMfaSucceededV1Input) error {
	return updateUserLoginV1(updateUserLoginV1Input{
		Db: opts.Db,
		Id: opts.Id,
		FieldsToSet: map[string]any{
			"is_pending_mfa": false,
		},
	})
}

type SetUserLoginStatusV1Input struct {
	Db *sql.DB

	Id     string
	Status string
}

func SetUserLoginStatusV1(opts SetUserLoginStatusV1Input) error {
	return updateUserLoginV1(updateUserLoginV1Input{
		Db: opts.Db,
		Id: opts.Id,
		FieldsToSet: map[string]any{
			"status": opts.Status,
		},
	})
}
