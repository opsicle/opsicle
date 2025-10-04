package models

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type OrgRolePermissions []OrgRolePermission
type OrgRolePermission struct {
	Id       *string  `json:"id"`
	OrgRole  *OrgRole `json:"orgRole"`
	Resource Resource `json:"resource"`
	Allows   Action   `json:"allows"`
	Denys    Action   `json:"denys"`
}

type CreateOrgRolePermissionV1Input struct {
	Allows   Action
	Denys    Action
	Resource Resource
	DatabaseConnection
}

func (or *OrgRole) CreatePermissionV1(opts CreateOrgRolePermissionV1Input) error {
	orgRolePermissionId := uuid.NewString()
	insertMap := map[string]any{
		"id":          orgRolePermissionId,
		"allows":      uint(opts.Allows),
		"denys":       uint(opts.Denys),
		"resource":    string(opts.Resource),
		"org_role_id": or.GetId(),
	}
	fieldNames, fieldValues, fieldPlaceholders, err := parseInsertMap(insertMap)
	if err != nil {
		return fmt.Errorf("failed to parse insert map: %w", err)
	}
	if err := executeMysqlInsert(mysqlQueryInput{
		Db:       opts.Db,
		FnSource: "models.OrgRole.CreatePermissionV1",
		Stmt: fmt.Sprintf(
			`INSERT INTO org_role_permissions (%s) VALUES (%s)`,
			strings.Join(fieldNames, ", "),
			strings.Join(fieldPlaceholders, ", "),
		),
		Args:         fieldValues,
		RowsAffected: oneRowAffected,
	}); err != nil {
		return fmt.Errorf("failed to insert org role permissions: %w", err)
	}
	or.Permissions = append(or.Permissions, OrgRolePermission{
		Id:       &orgRolePermissionId,
		Allows:   Action(insertMap["allows"].(uint)),
		Denys:    Action(insertMap["denys"].(uint)),
		Resource: opts.Resource,
		OrgRole:  or,
	})
	return nil
}
