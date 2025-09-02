package models

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type UpdateUserFieldsV1 struct {
	Db *sql.DB

	FieldsToSet map[string]any
}

func (u *User) UpdateFieldsV1(opts UpdateUserFieldsV1) error {
	if u.Id == nil {
		return fmt.Errorf("missing id")
	} else if _, err := uuid.Parse(*u.Id); err != nil {
		return fmt.Errorf("invalid id")
	}
	sqlArgs := []any{}
	fieldNames := []string{}
	fieldsToSet := []string{}
	for field, value := range opts.FieldsToSet {
		fieldNames = append(fieldNames, field)
		switch v := value.(type) {
		case string:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, v)
		case []byte:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, string(v))
		default:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, fmt.Sprintf("%v", v))
		}
	}
	return executeMysqlUpdate(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(`
			UPDATE users
				SET %s
				WHERE id = ?
			`,
			strings.Join(fieldsToSet, ", "),
		),
		Args: append(sqlArgs, *u.Id),
		FnSource: fmt.Sprintf(
			"models.OrgUser.UpdateFieldsV1['%s']",
			strings.Join(fieldNames, "','"),
		),
	})
}
