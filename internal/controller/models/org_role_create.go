package models

import (
	"fmt"
	"opsicle/internal/validate"
	"strings"

	"github.com/google/uuid"
)

const (
	DefaultOrgRoleName = "Administrator (Default)"
)

type CreateOrgRoleV1Input struct {
	UserId   string `json:"userId"`
	RoleName string `json:"roleName"`
	DatabaseConnection
}

func (o *Org) CreateRoleV1(opts CreateOrgRoleV1Input) (*OrgRole, error) {
	if o.Id == nil {
		return nil, fmt.Errorf("missing org id: %w", ErrorIdRequired)
	} else if err := validate.Uuid(*o.Id); err != nil {
		return nil, fmt.Errorf("invalid org id: %w", ErrorInvalidInput)
	}

	orgRoleId := uuid.NewString()

	insertMap := map[string]any{
		"created_by": opts.UserId,
		"id":         orgRoleId,
		"name":       opts.RoleName,
		"org_id":     o.GetId(),
	}
	fieldNames, fieldValues, fieldValuePlaceholders, err := parseInsertMap(insertMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse insert map: %w", errorInputValidationFailed)
	}

	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(
			`INSERT INTO org_roles (%s) VALUES (%s)`,
			strings.Join(fieldNames, ", "),
			strings.Join(fieldValuePlaceholders, ", "),
		),
		Args: fieldValues,
	}); err != nil {
		return nil, fmt.Errorf("failed to create role[%s]: %w", opts.RoleName, err)
	}

	return &OrgRole{
		Id:        &orgRoleId,
		OrgId:     o.Id,
		Name:      opts.RoleName,
		CreatedBy: &User{Id: &opts.UserId},
	}, nil
}
