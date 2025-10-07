package models

import (
	"database/sql"
	"fmt"
)

// ListTemplatesV1 retrieves templates associated with the organisation.
func (o *Org) ListTemplatesV1(opts DatabaseConnection) ([]Template, error) {
	if err := o.assertIdDefined(); err != nil {
		return nil, err
	}

	templates := []Template{}
	if err := executeMysqlSelects(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT
				at.id,
				at.name,
				at.description,
				at.created_at,
				at.created_by,
				at.last_updated_at,
				at.last_updated_by,
				atv.version,
				atv.content
			FROM templates at
			JOIN template_versions atv ON atv.template_id = at.id AND atv.version = at.version
			JOIN template_orgs ato ON ato.template_id = at.id
			WHERE ato.org_id = ?
			ORDER BY at.last_updated_at DESC
		`,
		Args:     []any{o.GetId()},
		FnSource: "models.Org.ListTemplatesV1",
		ProcessRows: func(r *sql.Rows) error {
			template := Template{CreatedBy: &User{}, LastUpdatedBy: &User{}}
			if err := r.Scan(
				&template.Id,
				&template.Name,
				&template.Description,
				&template.CreatedAt,
				&template.CreatedBy.Id,
				&template.LastUpdatedAt,
				&template.LastUpdatedBy.Id,
				&template.Version,
				&template.Content,
			); err != nil {
				return err
			}
			if template.CreatedBy.Id != nil {
				if err := template.CreatedBy.LoadByIdV1(opts); err != nil {
					return fmt.Errorf("failed to load user[%s]: %w", template.CreatedBy.GetId(), err)
				}
			} else {
				template.CreatedBy = nil
			}
			if template.LastUpdatedBy.Id != nil {
				if err := template.LastUpdatedBy.LoadByIdV1(opts); err != nil {
					return fmt.Errorf("failed to load user[%s]: %w", template.LastUpdatedBy.GetId(), err)
				}
			} else {
				template.LastUpdatedBy = nil
			}
			templates = append(templates, template)
			return nil
		},
	}); err != nil {
		return nil, err
	}

	o.Templates = templates
	return templates, nil
}
