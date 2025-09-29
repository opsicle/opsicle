package models

import (
	"database/sql"
	"fmt"
)

func (t *Template) GetAutomationParamsV1(opts DatabaseConnection) (*Automation, error) {
	if err := t.validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	} else if err := t.assertVersion(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	automation := &Automation{
		TemplateId:        t.GetId(),
		TemplateVersion:   t.GetVersion(),
		TemplateCreatedBy: &User{},
	}
	if err := executeMysqlSelect(mysqlQueryInput{
		Db:       opts.Db,
		FnSource: "models.Template.GetAutomationParamsV1",
		Stmt: `
			SELECT
				content,
				created_by
			FROM template_versions
			WHERE
				template_id = ?
				AND version = ?
		`,
		Args: []any{*t.Id, *t.Version},
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(
				&automation.TemplateContent,
				&automation.TemplateCreatedBy.Id,
			)
		},
	}); err != nil {
		return nil, err
	}

	return automation, nil
}
