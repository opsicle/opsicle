package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"opsicle/internal/validate"
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

type AssignOrgRoleV1Input struct {
	DatabaseConnection

	OrgId      string
	UserId     string
	AssignedBy *string
}

func (or OrgRole) AssignUserV1(opts AssignOrgRoleV1Input) error {
	if opts.Db == nil {
		return fmt.Errorf("missing db connection: %w", errorInputValidationFailed)
	}
	if or.Id == nil {
		return fmt.Errorf("org role id undefined: %w", errorInputValidationFailed)
	}
	if err := validate.Uuid(*or.Id); err != nil {
		return fmt.Errorf("invalid org role id: %w", errorInputValidationFailed)
	}
	if opts.OrgId == "" {
		return fmt.Errorf("org id undefined: %w", errorInputValidationFailed)
	} else if err := validate.Uuid(opts.OrgId); err != nil {
		return fmt.Errorf("invalid org id: %w", errorInputValidationFailed)
	}
	if opts.UserId == "" {
		return fmt.Errorf("user id undefined: %w", errorInputValidationFailed)
	} else if err := validate.Uuid(opts.UserId); err != nil {
		return fmt.Errorf("invalid user id: %w", errorInputValidationFailed)
	}

	insertMap := map[string]any{
		"user_id":     opts.UserId,
		"org_id":      opts.OrgId,
		"org_role_id": or.GetId(),
		"assigned_by": DatabaseFunction("NULL"),
	}
	if opts.AssignedBy != nil && *opts.AssignedBy != "" {
		if err := validate.Uuid(*opts.AssignedBy); err != nil {
			return fmt.Errorf("invalid assigned by id: %w", errorInputValidationFailed)
		}
		insertMap["assigned_by"] = *opts.AssignedBy
	}

	fieldNames, fieldValues, fieldPlaceholders, err := parseInsertMap(insertMap)
	if err != nil {
		return fmt.Errorf("failed to parse insert map: %w", err)
	}

	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(
			`INSERT INTO org_user_roles (%s) VALUES (%s)`,
			strings.Join(fieldNames, ", "),
			strings.Join(fieldPlaceholders, ", "),
		),
		Args: fieldValues,
	}); err != nil {
		return fmt.Errorf("failed to assign role[%s] to user[%s]: %w", or.GetId(), opts.UserId, err)
	}
	return nil
}
