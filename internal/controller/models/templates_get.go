package models

import (
	"database/sql"
	"fmt"
)

type GetTemplateV1Opts struct {
	Db           *sql.DB
	TemplateId   *string
	TemplateName *string
	UserId       string
}

func GetTemplateV1(opts GetTemplateV1Opts) (*Template, error) {
	fieldSelector := "at.id"
	selectorValue := ""
	if opts.TemplateName != nil {
		fieldSelector = "at.name"
		selectorValue = *opts.TemplateName
	} else if opts.TemplateId != nil {
		selectorValue = *opts.TemplateId
	}
	output := Template{}
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(`
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
				AND %s = ?
			ORDER BY atv.version DESC
		`, fieldSelector),
		Args: []any{
			opts.UserId,
			selectorValue,
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
