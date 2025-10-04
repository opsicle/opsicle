package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Org struct {
	// Id is a UUID that identifies the orgainsation uniquely
	Id *string `json:"id"`

	// Name is the display name of the organisation
	Name string `json:"name"`

	// Code is the shortcode for the organisation and has to be unique
	Code string `json:"code"`

	// Type defines the type of organisation
	Type string `json:"type"`

	// Icon optionally contains a URL/URI for the organisation's favicon
	Icon *string `json:"icon"`

	// Logo optionally contains a URL/URI for the organisation's logo
	Logo *string `json:"logo"`

	// Motd optionally contains a markdown text that the organisation
	// uses as a banner for all their users
	Motd *string `json:"motd"`

	// IsUsingExternalDatabase indicates whether the organisation data
	// is hosted on a separate database instance that is not the
	// shared database
	IsUsingExternalDatabase bool `json:"isUsingExternalDatabase"`

	// IsUsingTenantedDatabase indicates whether the organisation data
	// is hosted on a separate database schema in the shared database
	IsUsingTenantedDatabase bool `json:"isUsingTenantedDatabase"`

	// CreatedAt defines when the organisation was created
	CreatedAt time.Time `json:"createdAt"`

	// CreatedBy defines who the organisation was created by
	CreatedBy *User `json:"createdBy"`

	// CreatedAt defines when the organisation was last updated
	UpdatedAt *time.Time `json:"updatedAt"`

	// IsDeleted defines whether the organisation is scheduled for
	// deletion but pending any legal holds
	IsScheduledForDeletion bool `json:"isScheduledForDeletion"`

	// IsDeleted defines whether the organisation is scheduled for
	// deletion but pending any legal holds
	IsDeleted bool `json:"isDeleted"`

	// DeletedAt defines when the organisation was actually deleted
	DeletedAt *time.Time `json:"deletedAt"`

	// IsDisabled defines whether the organisation activities should
	// be paused
	IsDisabled bool `json:"isDisabled"`

	// DisabledAt defines the time when the organisation was disabled,
	// logs will be in the audit logs
	DisabledAt *time.Time `json:"disabledAt"`

	// UserCount stores the number of users registered to the organisation
	UserCount *int `json:"userCount"`

	// MemberType defines the type of membership of the current user, only
	// available when the organisation was queried as part of a user's
	// request regarding which organisations they belong to
	MemberType *string `json:"memberType"`

	// JoinedAt is an optionally available field for when a user requests
	// for organisations they belong to - this will be used as the timestamp
	// when the user joined the org
	JoinedAt *time.Time `json:"joinedAt"`

	// Roles contains the roles of this organisation
	Roles OrgRoles `json:"roles"`

	// Users contains the user sof this organisation when loaded
	Users OrgUsers `json:"users"`
}

func (o Org) assertIdDefined() error {
	if o.Id == nil {
		return fmt.Errorf("id undefined: %w", errorInputValidationFailed)
	} else if _, err := uuid.Parse(*o.Id); err != nil {
		return fmt.Errorf("id not uuid: %w", errorInputValidationFailed)
	}
	return nil
}

type AddUserToOrgV1 struct {
	Db *sql.DB

	UserId     string
	MemberType string
}

func (o *Org) AddUserV1(opts AddUserToOrgV1) error {
	if err := o.assertIdDefined(); err != nil {
		return err
	}
	memberType := TypeOrgMember
	if opts.MemberType != "" {
		if _, ok := OrgMemberTypeMap[opts.MemberType]; ok {
			memberType = OrgMemberType(opts.MemberType)
		}
	}
	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			INSERT INTO org_users(
				org_id,
				user_id,
				type
			) VALUES (?, ?, ?)
		`,
		Args: []any{
			*o.Id,
			opts.UserId,
			memberType,
		},
		FnSource:     "models.Org.AddUserV1",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return err
	}
	return nil
}

func (o Org) GetId() string {
	return *o.Id
}

func (o Org) GetAdminsV1(opts DatabaseConnection) ([]OrgUser, error) {
	if err := o.assertIdDefined(); err != nil {
		return nil, err
	}
	results := []OrgUser{}
	if err := executeMysqlSelects(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT 
				u.email,
				u.id,
				o.id,
				o.code,
				o.name,
				ou.type,
				ou.joined_at
				FROM org_users ou
					JOIN orgs o ON o.id = ou.org_id
					JOIN users u ON u.id = ou.user_id
				WHERE
					ou.org_id = ? AND ou.type = ?
		`,
		Args:     []any{*o.Id, TypeOrgAdmin},
		FnSource: "models.Org.GetAdminsV1",
		ProcessRows: func(r *sql.Rows) error {
			orgUser := NewOrgUser()
			if err := r.Scan(
				&orgUser.User.Email,
				&orgUser.User.Id,
				&orgUser.Org.Id,
				&orgUser.Org.Code,
				&orgUser.Org.Name,
				&orgUser.MemberType,
				&orgUser.JoinedAt,
			); err != nil {
				return err
			}
			results = append(results, orgUser)
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("models.Org.GetAdminsV1: failed to get admins: %w", err)
	}
	return results, nil
}

type GetOrgUserV1Opts struct {
	Db *sql.DB

	UserId string
}

// GetUserV1 retrieves the user as the organisation understands it
// based on the provided `UserId“
func (o *Org) GetUserV1(opts GetOrgUserV1Opts) (*OrgUser, error) {
	if err := o.assertIdDefined(); err != nil {
		return nil, err
	}
	userInstance := NewOrgUser()
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT
				users.email,
				users.id,
				orgs.id,
				orgs.code,
				orgs.name,
				org_users.type,
				org_users.joined_at
				FROM org_users
					JOIN orgs ON orgs.id = org_users.org_id
					JOIN users ON users.id = org_users.user_id
				WHERE
					org_id = ? AND user_id = ?
		`,
		Args:     []any{*o.Id, opts.UserId},
		FnSource: "models.Org.GetUserV1",
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(
				&userInstance.User.Email,
				&userInstance.User.Id,
				&userInstance.Org.Id,
				&userInstance.Org.Code,
				&userInstance.Org.Name,
				&userInstance.MemberType,
				&userInstance.JoinedAt,
			)
		},
	}); err != nil {
		return nil, err
	}
	return &userInstance, nil
}

type GetRoleCountV1Opts struct {
	Db *sql.DB

	Role OrgMemberType
}

func (o Org) GetRoleCountV1(opts GetRoleCountV1Opts) (int, error) {
	var roleCount int
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
		SELECT
			COUNT(*)
			FROM org_users
			WHERE
				org_id = ?
				AND type = ?
	`,
		Args:     []any{*o.Id, opts.Role},
		FnSource: "models.Org.GetRoleCountV1",
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(&roleCount)
		},
	}); err != nil {
		return -1, err
	}
	return roleCount, nil
}

type InviteOrgUserV1Opts struct {
	Db *sql.DB

	AcceptorId     *string
	AcceptorEmail  *string
	InviterId      string
	JoinCode       string
	MembershipType string
}

type InviteUserV1Output struct {
	InvitationId   string `json:"invitationId"`
	IsExistingUser bool   `json:"isExistingUser"`
}

func (o *Org) InviteUserV1(opts InviteOrgUserV1Opts) (*InviteUserV1Output, error) {
	if _, ok := OrgMemberTypeMap[opts.MembershipType]; !ok {
		opts.MembershipType = string(TypeOrgMember)
	}
	invitationId := uuid.NewString()

	sqlInserts := []string{
		"id",
		"org_id",
		"inviter_id",
		"join_code",
		"type",
	}
	sqlArgs := []any{
		invitationId,
		o.GetId(),
		opts.InviterId,
		opts.JoinCode,
		opts.MembershipType,
	}
	isExistingUser := false
	if opts.AcceptorId != nil {
		sqlInserts = append(sqlInserts, "acceptor_id")
		sqlArgs = append(sqlArgs, *opts.AcceptorId)
		isExistingUser = true
	} else if opts.AcceptorEmail != nil {
		sqlInserts = append(sqlInserts, "acceptor_email")
		sqlArgs = append(sqlArgs, *opts.AcceptorEmail)
	} else {
		return nil, fmt.Errorf("failed to receive either acceptor email or id: %w", ErrorInvalidInput)
	}
	sqlPlaceholders := []string{}
	for range len(sqlInserts) {
		sqlPlaceholders = append(sqlPlaceholders, "?")
	}
	sqlStmt := fmt.Sprintf(
		"INSERT INTO org_user_invitations(%s) VALUES (%s)",
		strings.Join(sqlInserts, ", "),
		strings.Join(sqlPlaceholders, ", "),
	)

	if err := executeMysqlInsert(mysqlQueryInput{
		Db:           opts.Db,
		Stmt:         sqlStmt,
		Args:         sqlArgs,
		FnSource:     "models.Org.InviteUserV1",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
	}

	return &InviteUserV1Output{
		InvitationId:   invitationId,
		IsExistingUser: isExistingUser,
	}, nil
}

func (o *Org) ListRolesV1(opts DatabaseConnection) (OrgRoles, error) {
	if err := o.assertIdDefined(); err != nil {
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
				u.email,
				orp.id,
				orp.resource,
				orp.allows,
				orp.denys
			FROM org_roles orls
				LEFT JOIN users u ON u.id = orls.created_by
				LEFT JOIN org_role_permissions orp ON orp.org_role_id = orls.id
			WHERE
				orls.org_id = ?
			ORDER BY
				orls.created_at ASC,
				orls.name ASC,
				orp.resource ASC
		`,
		Args:     []any{*o.Id},
		FnSource: "models.Org.ListRolesV1",
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
					OrgId:       o.Id,
					Name:        roleName,
					CreatedAt:   roleCreatedAt,
					Permissions: make(OrgRolePermissions, 0),
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

func (o *Org) ListUsersV1(opts DatabaseConnection) ([]OrgUser, error) {
	if err := o.assertIdDefined(); err != nil {
		return nil, err
	}
	output := []OrgUser{}
	if err := executeMysqlSelects(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT
				ou.joined_at,
				ou.type,
				o.code,
				o.id,
				o.name,
				u.email,
				u.id,
				u.type
				FROM org_users ou
					JOIN users u ON ou.user_id = u.id
					JOIN orgs o ON ou.org_id = o.id
				WHERE
					org_id = ?
		`,
		Args:     []any{*o.Id},
		FnSource: "models.Org.ListUsersV1",
		ProcessRows: func(r *sql.Rows) error {
			orgUser := NewOrgUser()
			if err := r.Scan(
				&orgUser.JoinedAt,
				&orgUser.MemberType,
				&orgUser.Org.Code,
				&orgUser.Org.Id,
				&orgUser.Org.Name,
				&orgUser.User.Email,
				&orgUser.User.Id,
				&orgUser.User.Type,
			); err != nil {
				return err
			}
			output = append(output, orgUser)
			return nil
		},
	}); err != nil {
		return nil, err
	}

	return output, nil
}

func (o *Org) LoadUserCountV1(opts DatabaseConnection) (int, error) {
	if err := o.assertIdDefined(); err != nil {
		return -1, err
	}
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT COUNT(*) AS 'count'
				FROM org_users
				WHERE
					org_id = ?
		`,
		Args:     []any{*o.Id},
		FnSource: "models.Org.LoadUserCountV1",
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(&o.UserCount)
		},
	}); err != nil {
		return -1, err
	}
	return *o.UserCount, nil
}
