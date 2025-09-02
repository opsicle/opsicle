package models

import (
	"database/sql"
)

type ListUserOrgsV1Opts struct {
	Db *sql.DB

	UserId string
}

// ListUserOrgsV1 returns an organisation given either it's ID or code;
// when no organisation is found, returns nil for both return values
func ListUserOrgsV1(opts ListUserOrgsV1Opts) ([]Org, error) {
	output := []Org{}
	if err := executeMysqlSelects(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT 
				o.id,
				o.name,
				o.created_at,
				o.last_updated_at,
				o.is_deleted,
				o.deleted_at,
				o.is_disabled,
				o.disabled_at,
				o.code,
				o.type as org_type,
				ou.type as member_type,
				ou.joined_at
				FROM orgs o
					JOIN org_users ou ON o.id = ou.org_id
				WHERE ou.user_id = ?
		`,
		Args:     []any{opts.UserId},
		FnSource: "models.ListUserOrgsV1",
		ProcessRows: func(r *sql.Rows) error {
			org := Org{}
			if err := r.Scan(
				&org.Id,
				&org.Name,
				&org.CreatedAt,
				&org.UpdatedAt,
				&org.IsDeleted,
				&org.DeletedAt,
				&org.IsDisabled,
				&org.DisabledAt,
				&org.Code,
				&org.Type,
				&org.MemberType,
				&org.JoinedAt,
			); err != nil {
				return err
			}
			output = append(output, org)
			return nil
		},
	}); err != nil {
		return nil, err
	}
	return output, nil
}
