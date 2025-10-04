package models

import (
	"database/sql"
	"time"
)

type OrgRoles []OrgRole
type OrgRole struct {
	Id            *string   `json:"id"`
	OrgId         *string   `json:"orgId"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"createdAt"`
	CreatedBy     *User     `json:"createdBy"`
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`

	Permissions OrgRolePermissions `json:"permissions"`
}

func (or *OrgRole) CanV1(opts DatabaseConnection, do Action, on Resource) bool {
	can := false
	if err := executeMysqlSelect(mysqlQueryInput{
		FnSource: "models.OrgRole.CanV1",
		Db:       opts.Db,
		Stmt: `
			SELECT 
				TRUE
				FROM org_role or
				JOIN org_role_permissions orp ON orp.role_id = or.id
			WHERE
				orp.allows & ? != 0
				AND orp.resource = ?
				AND or.id = ?
		`,
		Args: []any{
			do,
			on,
			*or.Id,
		},
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(&can)
		},
	}); err != nil {
		return false
	}
	return can
}

func (or *OrgRole) GetId() string {
	return *or.Id
}
