package models

import "database/sql"

func (u *User) ListTemplatesV1(opts DatabaseConnection) ([]AutomationTemplate, error) {
	output := []AutomationTemplate{}
	if err := executeMysqlSelects(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
		  SELECT 
				at.id,
				at.name,
				at.description,
				atv.version,
				atv.content,
				u.id,
				u.email
				FROM automation_templates at
				JOIN automation_template_versions atv ON atv.automation_template_id = at.id AND atv.version = at.version
				JOIN automation_template_users atu ON atu.automation_template_id = at.id
				JOIN users u ON atu.user_id = u.id
			WHERE
				atu.user_id = ?
		`,
		Args: []any{
			u.GetId(),
		},
		ProcessRows: func(r *sql.Rows) error {
			template := AutomationTemplate{CreatedBy: User{}}
			if err := r.Scan(
				&template.Id,
				&template.Name,
				&template.Description,
				&template.Version,
				&template.Content,
				&template.CreatedBy.Id,
				&template.CreatedBy.Email,
			); err != nil {
				return err
			}
			output = append(output, template)
			return nil
		},
	}); err != nil {
		return nil, err
	}
	return output, nil
}
