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
	fieldNames, fieldsToSet, sqlArgs, err := parseUpdateMap(opts.FieldsToSet)
	if err != nil {
		return fmt.Errorf("failed to parse update map: %w", err)
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
