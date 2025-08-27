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
	fieldsToSet := []string{}
	for field, value := range opts.FieldsToSet {
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
	sqlStmt := fmt.Sprintf(`
	UPDATE org_users
		SET %s
		WHERE org_id = ? AND user_id = ?`, strings.Join(fieldsToSet, ", "))
	sqlArgs = append(sqlArgs, ou.OrgId, ou.UserId)

	return executeMysqlUpdate(mysqlQueryInput{
		Db:           opts.Db,
		Stmt:         sqlStmt,
		Args:         sqlArgs,
		RowsAffected: oneRowAffected,
		FnSource:     "models.OrgUser.UpdateFieldsV1",
	})
	return nil
}
