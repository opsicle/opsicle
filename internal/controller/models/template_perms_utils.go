package models

import (
	"database/sql"
	"fmt"
)

func (t *Template) CanUserUpdateV1(opts DatabaseConnection, userId string) (bool, error) {
	if t.Id == nil {
		return false, fmt.Errorf("%w: template id not specified", ErrorInvalidInput)
	}
	var canView, canUpdate bool
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT
				can_view,
				can_update
			FROM
				automation_template_users
			WHERE
				automation_template_id = ?
				AND user_id = ?
		`,
		Args: []any{
			*t.Id,
			userId,
		},
		FnSource: "models.Template.CanUserUpdate",
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(&canView, &canUpdate)
		},
	}); err != nil {
		if isMysqlNotFoundError(err) {
			return false, fmt.Errorf("%w: %w", ErrorNotFound, err)
		}
		return false, fmt.Errorf("%w: %w", ErrorGenericDatabaseIssue, err)
	}
	return canView && canUpdate, nil
}
