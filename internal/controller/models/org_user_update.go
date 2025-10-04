package models

import (
	"fmt"
	"strings"
)

func (ou *OrgUser) UpdateFieldsV1(opts UpdateFieldsV1) error {
	if err := ou.validate(); err != nil {
		return err
	}
	fieldNames, fieldsToSet, sqlArgs, err := parseUpdateMap(opts.FieldsToSet)
	if err != nil {
		return fmt.Errorf("failed to parse update map: %w", err)
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
		Args: append(sqlArgs, ou.Org.GetId(), ou.User.GetId()),
		FnSource: fmt.Sprintf(
			"models.OrgUser.UpdateFieldsV1['%s']",
			strings.Join(fieldNames, "','"),
		),
	})
}
