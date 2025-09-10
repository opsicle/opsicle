package models

import (
	"database/sql"
	"fmt"
)

func (t *AutomationTemplate) LoadVersionsV1(opts DatabaseConnection) error {
	if t.Id == nil {
		return fmt.Errorf("%w: template id not specified", ErrorInvalidInput)
	}
	t.Versions = []TemplateVersion{}
	userMap := map[string]User{}
	if err := executeMysqlSelects(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT
				version,
				content,
				created_at,
				created_by
			FROM
				automation_template_versions atv
			WHERE
				atv.automation_template_id = ?
		`,
		Args: []any{
			*t.Id,
		},
		FnSource: "models.AutomationTemplate.LoadVersionsV1",
		ProcessRows: func(r *sql.Rows) error {
			templateVersion := TemplateVersion{}
			if err := r.Scan(
				&templateVersion.Version,
				&templateVersion.Content,
				&templateVersion.CreatedAt,
				&templateVersion.CreatedBy.Id,
			); err != nil {
				return err
			}
			if templateVersion.CreatedBy.Id != nil {
				user, ok := userMap[*templateVersion.CreatedBy.Id]
				if !ok {
					if err := templateVersion.CreatedBy.LoadByIdV1(opts); err != nil {
						return fmt.Errorf("failed to load createdBy: %w", err)
					}
					userMap[*templateVersion.CreatedBy.Id] = templateVersion.CreatedBy
				} else {
					templateVersion.CreatedBy = user
				}
			}
			t.Versions = append(t.Versions, templateVersion)
			return nil
		},
	}); err != nil {
		return fmt.Errorf("%w: %w", ErrorGenericDatabaseIssue, err)
	}

	return nil
}
