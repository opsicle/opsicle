package models

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"opsicle/internal/auth"
	"opsicle/internal/validate"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OrgTokens []OrgToken

type OrgToken struct {
	Id             *string   `json:"id" yaml:"id"`
	Org            *Org      `json:"org" yaml:"org"`
	Name           string    `json:"name" yaml:"name"`
	Description    *string   `json:"description" yaml:"description"`
	CertificateB64 string    `json:"certificateB64" yaml:"certificateB64"`
	PrivateKeyB64  string    `json:"privateKeyB64" yaml:"privateKeyB64"`
	ApiKey         string    `json:"apiKey" yaml:"apiKey"`
	CreatedAt      time.Time `json:"createdAt" yaml:"createdAt"`
	CreatedBy      *User     `json:"createdBy" yaml:"createdBy"`
	LastUpdatedAt  time.Time `json:"lastUpdatedAt" yaml:"lastUpdatedAt"`
	LastUpdatedBy  *User     `json:"lastUpdatedBy" yaml:"lastUpdatedBy"`
	Role           *OrgRole  `json:"role" yaml:"role"`
}

func (ot OrgToken) GetId() string {
	if ot.Id == nil {
		return ""
	}
	return *ot.Id
}

func (ot OrgToken) GetOrg() *Org {
	if ot.Org == nil {
		return &Org{}
	}
	return ot.Org
}

func (ot OrgToken) GetRedacted() OrgToken {
	redacted := ot
	redacted.CertificateB64 = ""
	redacted.PrivateKeyB64 = ""
	redacted.ApiKey = ""
	if redacted.Role != nil {
		roleCopy := *redacted.Role
		if roleCopy.CreatedBy != nil {
			tmp := roleCopy.CreatedBy.GetRedacted()
			roleCopy.CreatedBy = &tmp
		}
		if len(roleCopy.Permissions) > 0 {
			perms := make(OrgRolePermissions, len(roleCopy.Permissions))
			for idx, perm := range roleCopy.Permissions {
				permCopy := perm
				permCopy.OrgRole = &roleCopy
				perms[idx] = permCopy
			}
			roleCopy.Permissions = perms
		}
		redacted.Role = &roleCopy
	}
	return redacted
}

func (ots OrgTokens) GetRedacted() OrgTokens {
	if len(ots) == 0 {
		return OrgTokens{}
	}
	output := make(OrgTokens, 0, len(ots))
	for _, token := range ots {
		redacted := token.GetRedacted()
		if redacted.CreatedBy != nil {
			user := redacted.CreatedBy.GetRedacted()
			redacted.CreatedBy = &user
		}
		if redacted.LastUpdatedBy != nil {
			user := redacted.LastUpdatedBy.GetRedacted()
			redacted.LastUpdatedBy = &user
		}
		if redacted.Role != nil {
			roleCopy := *redacted.Role
			if roleCopy.CreatedBy != nil {
				tmp := roleCopy.CreatedBy.GetRedacted()
				roleCopy.CreatedBy = &tmp
			}
			if len(roleCopy.Permissions) > 0 {
				perms := make(OrgRolePermissions, len(roleCopy.Permissions))
				for idx, perm := range roleCopy.Permissions {
					permCopy := perm
					permCopy.OrgRole = &roleCopy
					perms[idx] = permCopy
				}
				roleCopy.Permissions = perms
			}
			redacted.Role = &roleCopy
		}
		output = append(output, redacted)
	}
	return output
}

type CreateOrgTokenV1Input struct {
	DatabaseConnection

	// TokenId is specified as an input (this breaks the usual pattern) because
	// we need to use it to identify the token in-use. This ID is used as the reference
	// so that the ApiKey does not need to be stored in plaintext in our database and
	// we can store a hash of it that can be compared to the service requester instead
	TokenId        string
	Name           string
	Description    *string
	CertificatePem []byte
	PrivateKeyPem  []byte
	ApiKey         string
	CreatedBy      *string
	OrgRole        *OrgRole
}

func (o *Org) CreateTokenV1(opts CreateOrgTokenV1Input) (*OrgToken, error) {
	if err := o.assertIdDefined(); err != nil {
		return nil, err
	}
	if opts.Db == nil {
		return nil, fmt.Errorf("missing db connection: %w", errorInputValidationFailed)
	}
	if err := validate.Uuid(opts.TokenId); err != nil {
		return nil, fmt.Errorf("token id invalid: %w", errorInputValidationFailed)
	}
	if opts.Name == "" {
		return nil, fmt.Errorf("token name undefined: %w", errorInputValidationFailed)
	}
	if len(opts.CertificatePem) == 0 {
		return nil, fmt.Errorf("certificate pem undefined: %w", errorInputValidationFailed)
	}
	if len(opts.PrivateKeyPem) == 0 {
		return nil, fmt.Errorf("private key pem undefined: %w", errorInputValidationFailed)
	}
	if opts.ApiKey == "" {
		return nil, fmt.Errorf("api key undefined: %w", errorInputValidationFailed)
	}
	if opts.OrgRole == nil || opts.OrgRole.Id == nil {
		return nil, fmt.Errorf("org role undefined: %w", errorInputValidationFailed)
	}

	hashedApiKey, err := auth.HashPassword(opts.ApiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to hash api key: %w", err)
	}

	certificateB64 := base64.StdEncoding.EncodeToString(opts.CertificatePem)
	privateKeyB64 := base64.StdEncoding.EncodeToString(opts.PrivateKeyPem)

	insertMap := map[string]any{
		"id":      opts.TokenId,
		"name":    opts.Name,
		"api_key": hashedApiKey,
		"org_id":  o.GetId(),
	}
	if opts.Description != nil {
		insertMap["description"] = *opts.Description
	}
	if opts.CreatedBy != nil {
		insertMap["created_by"] = *opts.CreatedBy
		insertMap["last_updated_by"] = *opts.CreatedBy
	}

	fieldNames, fieldValues, fieldPlaceholders, err := parseInsertMap(insertMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse insert map: %w", err)
	}

	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(
			`INSERT INTO org_tokens (%s) VALUES (%s)`,
			strings.Join(fieldNames, ", "),
			strings.Join(fieldPlaceholders, ", "),
		),
		Args:         fieldValues,
		FnSource:     "models.Org.CreateTokenV1[org_tokens]",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
	}

	roleLinkId := uuid.NewString()
	roleInsertMap := map[string]any{
		"id":           roleLinkId,
		"org_role_id":  opts.OrgRole.GetId(),
		"org_token_id": opts.TokenId,
	}
	if opts.CreatedBy != nil {
		roleInsertMap["created_by"] = *opts.CreatedBy
		roleInsertMap["last_updated_by"] = *opts.CreatedBy
	}
	roleFieldNames, roleValues, rolePlaceholders, err := parseInsertMap(roleInsertMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse role insert map: %w", err)
	}
	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(
			`INSERT INTO org_token_roles (%s) VALUES (%s)`,
			strings.Join(roleFieldNames, ", "),
			strings.Join(rolePlaceholders, ", "),
		),
		Args:         roleValues,
		FnSource:     "models.Org.CreateTokenV1[org_token_roles]",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
	}

	now := time.Now()
	description := opts.Description
	var createdBy *User
	var lastUpdatedBy *User
	if opts.CreatedBy != nil {
		createdBy = &User{Id: opts.CreatedBy}
		lastUpdatedBy = createdBy
	}
	token := OrgToken{
		Id:             &opts.TokenId,
		Org:            o,
		Name:           opts.Name,
		Description:    description,
		CertificateB64: certificateB64,
		PrivateKeyB64:  privateKeyB64,
		ApiKey:         opts.ApiKey,
		CreatedAt:      now,
		CreatedBy:      createdBy,
		LastUpdatedAt:  now,
		LastUpdatedBy:  lastUpdatedBy,
		Role:           opts.OrgRole,
	}
	return &token, nil
}

type DeleteOrgTokenV1Input struct {
	DatabaseConnection

	TokenId string
}

func (o *Org) DeleteTokenV1(opts DeleteOrgTokenV1Input) error {
	if err := o.assertIdDefined(); err != nil {
		return err
	}
	if opts.Db == nil {
		return fmt.Errorf("missing db connection: %w", errorInputValidationFailed)
	}
	if opts.TokenId == "" {
		return fmt.Errorf("token id undefined: %w", errorInputValidationFailed)
	}
	if err := executeMysqlDelete(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			DELETE FROM org_tokens
			WHERE id = ? AND org_id = ?
		`,
		Args: []any{
			opts.TokenId,
			o.GetId(),
		},
		FnSource:     "models.Org.DeleteTokenV1",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return err
	}
	return nil
}

type ListOrgTokensV1Opts struct {
	DatabaseConnection
}

func (o *Org) ListTokensV1(opts ListOrgTokensV1Opts) (OrgTokens, error) {
	if err := o.assertIdDefined(); err != nil {
		return nil, err
	}
	if opts.Db == nil {
		return nil, fmt.Errorf("missing db connection: %w", errorInputValidationFailed)
	}
	tokens := OrgTokens{}
	if err := executeMysqlSelects(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT
				id,
				name,
				description,
				api_key,
				created_at,
				created_by,
				last_updated_at,
				last_updated_by
			FROM org_tokens
			WHERE org_id = ?
		`,
		Args:     []any{o.GetId()},
		FnSource: "models.Org.ListTokensV1",
		ProcessRows: func(r *sql.Rows) error {
			token := OrgToken{Org: o}
			var (
				id              string
				description     sql.NullString
				createdById     sql.NullString
				lastUpdatedById sql.NullString
			)
			if err := r.Scan(
				&id,
				&token.Name,
				&description,
				&token.ApiKey,
				&token.CreatedAt,
				&createdById,
				&token.LastUpdatedAt,
				&lastUpdatedById,
			); err != nil {
				return err
			}
			token.Id = &id
			if description.Valid {
				descCopy := description.String
				token.Description = &descCopy
			}
			if createdById.Valid {
				createdBy := createdById.String
				token.CreatedBy = &User{Id: &createdBy}
				token.CreatedBy.LoadByIdV1(DatabaseConnection{Db: opts.Db})
			}
			if lastUpdatedById.Valid {
				lastUpdatedBy := lastUpdatedById.String
				token.LastUpdatedBy = &User{Id: &lastUpdatedBy}
				token.LastUpdatedBy.LoadByIdV1(DatabaseConnection{Db: opts.Db})
			}
			tokens = append(tokens, token)
			return nil
		},
	}); err != nil {
		return nil, err
	}
	return tokens, nil
}

type GetOrgTokenByIdV1Opts struct {
	DatabaseConnection

	TokenId string
}

func (o *Org) GetTokenByIdV1(opts GetOrgTokenByIdV1Opts) (*OrgToken, error) {
	if err := o.assertIdDefined(); err != nil {
		return nil, err
	}
	if opts.Db == nil {
		return nil, fmt.Errorf("missing db connection: %w", errorInputValidationFailed)
	}
	if err := validate.Uuid(opts.TokenId); err != nil {
		return nil, fmt.Errorf("token id invalid: %w", errorInputValidationFailed)
	}
	var (
		token               *OrgToken
		role                *OrgRole
		rolePermissionsSeen = map[string]struct{}{}
		tokenDescription    sql.NullString
		tokenCreatedBy      sql.NullString
		tokenLastUpdatedBy  sql.NullString
		roleId              sql.NullString
		roleName            sql.NullString
		roleCreatedAt       sql.NullTime
		roleLastUpdatedAt   sql.NullTime
		roleCreatedById     sql.NullString
		roleCreatedByEmail  sql.NullString
		permissionId        sql.NullString
		permissionResource  sql.NullString
		permissionAllows    sql.NullInt64
		permissionDenys     sql.NullInt64
	)
	if err := executeMysqlSelects(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT
				ot.id,
				ot.org_id,
				ot.name,
				ot.description,
				ot.created_at,
				ot.created_by,
				ot.last_updated_at,
				ot.last_updated_by,
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
			FROM org_tokens ot
				LEFT JOIN org_token_roles otr ON otr.org_token_id = ot.id
				LEFT JOIN org_roles orls ON orls.id = otr.org_role_id
				LEFT JOIN users u ON u.id = orls.created_by
				LEFT JOIN org_role_permissions orp ON orp.org_role_id = orls.id
			WHERE
				ot.id = ?
				AND ot.org_id = ?
		`,
		Args: []any{
			opts.TokenId,
			o.GetId(),
		},
		FnSource: "models.Org.GetTokenByIdV1",
		ProcessRows: func(r *sql.Rows) error {
			var (
				tokenId            string
				tokenOrgId         string
				tokenName          string
				tokenCreatedAt     time.Time
				tokenLastUpdatedAt time.Time
			)
			if err := r.Scan(
				&tokenId,
				&tokenOrgId,
				&tokenName,
				&tokenDescription,
				&tokenCreatedAt,
				&tokenCreatedBy,
				&tokenLastUpdatedAt,
				&tokenLastUpdatedBy,
				&roleId,
				&roleName,
				&roleCreatedAt,
				&roleLastUpdatedAt,
				&roleCreatedById,
				&roleCreatedByEmail,
				&permissionId,
				&permissionResource,
				&permissionAllows,
				&permissionDenys,
			); err != nil {
				return err
			}
			if token == nil {
				token = &OrgToken{
					Id:            &tokenId,
					Org:           o,
					Name:          tokenName,
					CreatedAt:     tokenCreatedAt,
					LastUpdatedAt: tokenLastUpdatedAt,
				}
				if tokenDescription.Valid {
					description := tokenDescription.String
					token.Description = &description
				}
				if tokenCreatedBy.Valid {
					createdBy := tokenCreatedBy.String
					token.CreatedBy = &User{Id: &createdBy}
				}
				if tokenLastUpdatedBy.Valid {
					lastUpdatedBy := tokenLastUpdatedBy.String
					token.LastUpdatedBy = &User{Id: &lastUpdatedBy}
				}
			}
			if roleId.Valid {
				if role == nil {
					roleIdValue := roleId.String
					role = &OrgRole{
						Id:          &roleIdValue,
						OrgId:       &tokenOrgId,
						Permissions: OrgRolePermissions{},
					}
					if roleName.Valid {
						role.Name = roleName.String
					}
					if roleCreatedAt.Valid {
						role.CreatedAt = roleCreatedAt.Time
					}
					if roleLastUpdatedAt.Valid {
						role.LastUpdatedAt = roleLastUpdatedAt.Time
					} else if roleCreatedAt.Valid {
						role.LastUpdatedAt = roleCreatedAt.Time
					}
					if roleCreatedById.Valid {
						createdById := roleCreatedById.String
						user := &User{Id: &createdById}
						if roleCreatedByEmail.Valid {
							user.Email = roleCreatedByEmail.String
						}
						role.CreatedBy = user
					}
				}
				if permissionId.Valid {
					if _, exists := rolePermissionsSeen[permissionId.String]; !exists {
						rolePermissionsSeen[permissionId.String] = struct{}{}
						permissionIdValue := permissionId.String
						permission := OrgRolePermission{
							Id:       &permissionIdValue,
							OrgRole:  role,
							Resource: Resource(permissionResource.String),
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
			}
			return nil
		},
	}); err != nil {
		return nil, err
	}
	if token == nil {
		return nil, ErrorNotFound
	}
	if role != nil {
		token.Role = role
	}
	return token, nil
}
