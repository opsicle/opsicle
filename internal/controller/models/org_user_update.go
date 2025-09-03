package models

import (
	"database/sql"
	"fmt"
	"strings"
)

type UpdateOrgUserFieldsV1 struct {
	Db *sql.DB

	FieldsToSet map[string]any
}

func (ou *OrgUser) UpdateFieldsV1(opts UpdateOrgUserFieldsV1) error {
	if err := ou.validate(); err != nil {
		return err
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
		case bool:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, v)
		default:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, fmt.Sprintf("%v", v))
		}
	}
	return executeMysqlUpdate(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(`
			UPDATE org_users
				SET %s
				WHERE org_id = ? AND user_id = ?
			`,
			strings.Join(fieldsToSet, ", "),
		),
		Args: append(sqlArgs, ou.OrgId, ou.UserId),
		FnSource: fmt.Sprintf(
			"models.OrgUser.UpdateFieldsV1['%s']",
			strings.Join(fieldNames, "','"),
		),
	})
}
