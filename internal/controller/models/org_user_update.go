package models

import (
	"fmt"
	"reflect"
	"strings"
)

func (ou *OrgUser) UpdateFieldsV1(opts UpdateFieldsV1) error {
	if err := ou.validate(); err != nil {
		return err
	}
	sqlArgs := []any{}
	fieldNames := []string{}
	fieldsToSet := []string{}
	for field, value := range opts.FieldsToSet {
		fieldNames = append(fieldNames, field)
		switch v := value.(type) {
		case string, int, int32, int64, float32, float64, bool:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, v)
		case []byte:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, string(v))
		case DatabaseFunction:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = %s", field, v))
		default:
			valueType := reflect.TypeOf(v)
			return fmt.Errorf("field[%s] has invalid type '%s'", field, valueType.String())
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
