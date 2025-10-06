package models

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func NewOrgUser() OrgUser {
	return OrgUser{
		Org:  &Org{},
		User: &User{},
		Role: &OrgRole{},
	}
}

type OrgUsers []OrgUser
type OrgUser struct {
	JoinedAt   time.Time `json:"joinedAt"`
	MemberType string    `json:"memberType"`
	Org        *Org      `json:"org"`
	User       *User     `json:"user"`

	Role *OrgRole `json:"role"`
}

func (ou OrgUser) validate() error {
	if ou.Org == nil {
		return fmt.Errorf("org undefined")
	} else if ou.Org.Id == nil {
		return fmt.Errorf("org id undefined")
	} else if _, err := uuid.Parse(ou.Org.GetId()); err != nil {
		return fmt.Errorf("org id is not a uuid: %w", ErrorInvalidInput)
	}

	if ou.User == nil {
		return fmt.Errorf("user undefined")
	} else if ou.User.Id == nil {
		return fmt.Errorf("user id undefined")
	} else if _, err := uuid.Parse(ou.User.GetId()); err != nil {
		return fmt.Errorf("user id is not a uuid: %w", ErrorInvalidInput)
	}
	return nil
}

func (ou *OrgUser) ListRolesV1(opts DatabaseConnection) (OrgRoles, error) {
	if err := ou.validate(); err != nil {
		return nil, err
	}
	roleMap := map[string]*OrgRole{}
	roleOrder := []string{}
	rolePermissionsSeen := map[string]map[string]struct{}{}
	if err := executeMysqlSelects(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT
				orls.id,
				orls.name,
				orls.created_at,
				orls.last_updated_at,
				orls.created_by,
				created_by_user.email,
				orp.id,
				orp.resource,
				orp.allows,
				orp.denys
			FROM org_user_roles our
				JOIN org_roles orls ON orls.id = our.org_role_id
				LEFT JOIN users created_by_user ON created_by_user.id = orls.created_by
				LEFT JOIN org_role_permissions orp ON orp.org_role_id = orls.id
			WHERE
				our.org_id = ?
				AND our.user_id = ?
			ORDER BY
				orls.created_at ASC,
				orls.name ASC,
				orp.resource ASC
		`,
		Args:     []any{ou.Org.GetId(), ou.User.GetId()},
		FnSource: "models.OrgUser.ListRolesV1",
		ProcessRows: func(r *sql.Rows) error {
			var (
				roleId           string
				roleName         string
				roleCreatedAt    time.Time
				roleUpdatedAtRaw sql.NullTime
				createdById      sql.NullString
				createdByEmail   sql.NullString
				permissionId     sql.NullString
				permissionRes    sql.NullString
				permissionAllows sql.NullInt64
				permissionDenys  sql.NullInt64
			)
			if err := r.Scan(
				&roleId,
				&roleName,
				&roleCreatedAt,
				&roleUpdatedAtRaw,
				&createdById,
				&createdByEmail,
				&permissionId,
				&permissionRes,
				&permissionAllows,
				&permissionDenys,
			); err != nil {
				return err
			}
			role, ok := roleMap[roleId]
			if !ok {
				role = &OrgRole{
					Id:          &roleId,
					OrgId:       ou.Org.Id,
					Name:        roleName,
					CreatedAt:   roleCreatedAt,
					Permissions: OrgRolePermissions{},
				}
				role.LastUpdatedAt = roleCreatedAt
				if roleUpdatedAtRaw.Valid {
					role.LastUpdatedAt = roleUpdatedAtRaw.Time
				}
				if createdById.Valid {
					createdByIdValue := createdById.String
					role.CreatedBy = &User{Id: &createdByIdValue, Email: createdByEmail.String}
				}
				roleMap[roleId] = role
				roleOrder = append(roleOrder, roleId)
				rolePermissionsSeen[roleId] = map[string]struct{}{}
			}
			if permissionId.Valid {
				seen := rolePermissionsSeen[roleId]
				if _, exists := seen[permissionId.String]; !exists {
					seen[permissionId.String] = struct{}{}
					permissionIdValue := permissionId.String
					permission := OrgRolePermission{
						Id:       &permissionIdValue,
						OrgRole:  role,
						Resource: Resource(permissionRes.String),
					}
					if permissionAllows.Valid {
						permission.Allows = Action(uint(permissionAllows.Int64))
					}
					if permissionDenys.Valid {
						permission.Denys = Action(uint(permissionDenys.Int64))
					}
					role.Permissions = append(role.Permissions, permission)
				}
			}
			return nil
		},
	}); err != nil {
		return nil, err
	}
	output := make(OrgRoles, 0, len(roleOrder))
	for _, roleId := range roleOrder {
		if role := roleMap[roleId]; role != nil {
			output = append(output, *role)
		}
	}
	return output, nil
}
