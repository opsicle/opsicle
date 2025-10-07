package models

import (
	"database/sql"
	"fmt"
)

type GetOrgTemplateV1Opts struct {
	Db           *sql.DB
	OrgId        string
	TemplateId   *string
	TemplateName *string
	UserId       string
}

func GetOrgTemplateV1(opts GetOrgTemplateV1Opts) (*Template, error) {
	if opts.OrgId == "" {
		return nil, fmt.Errorf("missing organization id: %w", errorInputValidationFailed)
	}
	fieldSelector := "id"
	selectorValue := ""
	if opts.TemplateName != nil {
		fieldSelector = "name"
		selectorValue = *opts.TemplateName
	} else if opts.TemplateId != nil {
		selectorValue = *opts.TemplateId
	}
	args := []any{selectorValue, opts.OrgId}
	statement := fmt.Sprintf(`
	  SELECT 
	        t.id,
	        t.name,
	        t.description,
	        atv.content,
	        atv.version
	    FROM templates t
				JOIN template_versions atv ON atv.template_id = t.id
				JOIN template_orgs ato ON ato.template_id = t.id
	    WHERE
				t.%s = ?
				AND ato.org_id = ?
	    ORDER BY atv.version DESC
	    LIMIT 1
	`, fieldSelector)
	output := Template{}
	if err := executeMysqlSelect(mysqlQueryInput{
		Db:       opts.Db,
		Stmt:     statement,
		Args:     args,
		FnSource: "models.GetOrgTemplateV1",
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
