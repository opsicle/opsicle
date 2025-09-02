package models

import (
	"database/sql"
	"fmt"
)

type GetOrgV1Opts struct {
	Db *sql.DB

	Id   *string
	Code *string
}

// GetOrgV1 returns an organisation given either it's ID or code;
// when no organisation is found, returns nil for both return values
func GetOrgV1(opts GetOrgV1Opts) (*Org, error) {
	if opts.Id == nil && opts.Code == nil {
		return nil, fmt.Errorf("models.GetOrgV1: either the org id or its code has to be specified")
	}
	selectorField := "id"
	selectorValue := ""
	if opts.Id != nil {
		selectorValue = *opts.Id
	} else if opts.Code != nil {
		selectorField = "code"
		selectorValue = *opts.Code
	}

	var output Org
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(`
			SELECT 
				id,
				name,
				created_at,
				last_updated_at,
				is_deleted,
				deleted_at,
				is_disabled,
				disabled_at,
				code,
				type,
				motd
				FROM orgs
				WHERE %s = ?`,
			selectorField,
		),
		Args:     []any{selectorValue},
		FnSource: "models.GetOrgV1",
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(
				&output.Id,
				&output.Name,
				&output.CreatedAt,
				&output.UpdatedAt,
				&output.IsDeleted,
				&output.DeletedAt,
				&output.IsDisabled,
				&output.DisabledAt,
				&output.Code,
				&output.Type,
				&output.Motd,
			)
		},
	}); err != nil {
		return nil, err
	}
	return &output, nil
}
