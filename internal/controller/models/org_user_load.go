package models

import (
	"database/sql"
)

// LoadV1 loads an organisation user based on the `UserId` and `OrgId`,
// if these are empty or not UUIDs, this function will return an
// ErrorInvalidInput error
func (ou *OrgUser) LoadV1(opts DatabaseConnection) error {
	if err := ou.validate(); err != nil {
		return err
	}
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT 
				ou.joined_at,
				ou.type,
				u.email,
				u.type,
				o.code,
				o.name
				FROM org_users ou
					JOIN users u ON ou.user_id = u.id
					JOIN orgs o ON ou.org_id = o.id
				WHERE 
					ou.org_id = ?
					AND ou.user_id = ?
		`,
		Args:     []any{ou.Org.GetId(), ou.User.GetId()},
		FnSource: "models.OrgUser.LoadV1",
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(
				&ou.JoinedAt,
				&ou.MemberType,
				&ou.User.Email,
				&ou.User.Type,
				&ou.Org.Code,
				&ou.Org.Name,
			)
		},
	}); err != nil {
		return err
	}
	return nil
}
