package models

import "database/sql"

type GetTemplateV1Opts struct {
	Db           *sql.DB
	TemplateName string
	UserId       string
}

func GetTemplateV1(opts GetTemplateV1Opts) (*Template, error) {
	output := Template{}
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
		  SELECT 
				at.id,
				at.name,
				at.description,
				atv.content,
				atv.version
				FROM automation_templates at
				JOIN automation_template_versions atv ON atv.automation_template_id = at.id
				JOIN automation_template_users atu ON atu.automation_template_id = at.id
			WHERE
				atu.user_id = ?
				AND at.name = ?
			ORDER BY atv.version DESC
		`,
		Args: []any{
			opts.UserId,
			opts.TemplateName,
		},
		FnSource: "models.GetTemplateV1",
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(
				&output.Id,
				&output.Name,
				&output.Description,
				&output.Content,
				&output.Version,
			)
		},
	}); err != nil {
		return nil, err
	}
	return &output, nil
}
