package models

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type CreateOrgV1Opts struct {
	Db *sql.DB

	// Name defines the name of the organisation as displayed to users
	Name string

	// Code defines the codeword to associate with the new
	// organisation; this has to be unique across all organisations
	Code string

	// Type defines the type of the organisation to create
	Type OrgType

	// UserId is the ID of the user creating the organisation,
	// which will be assigned as the administrator of the newly
	// created organisation
	UserId string
}

func CreateOrgV1(opts CreateOrgV1Opts) (string, error) {
	orgUuid := uuid.NewString()

	stmt, err := opts.Db.Prepare(`
	INSERT INTO orgs(
		id,
		name,
		code,
		type,
		created_by
	) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return "", fmt.Errorf("models.CreateOrgV1: failed to prepare insert statement for orgs: %w", err)
	}

	_, err = stmt.Exec(
		orgUuid,
		opts.Name,
		opts.Code,
		string(opts.Type),
		opts.UserId,
	)
	if err != nil {
		if isMysqlDuplicateError(err) {
			return "", fmt.Errorf("models.CreateOrgV1: failed to create an org with a duplicate codeword: %w", ErrorDuplicateEntry)
		}
		return "", fmt.Errorf("models.CreateOrgV1: failed to execute insert statement for orgs: %w", err)
	}

	stmt, err = opts.Db.Prepare(`
	INSERT INTO org_users(
		org_id,
		user_id,
		type
	) VALUES (?, ?, ?)
	`)
	if err != nil {
		return "", fmt.Errorf("models.CreateOrgV1: failed to prepare insert statement for org_users: %w", err)
	}

	_, err = stmt.Exec(
		orgUuid,
		opts.UserId,
		TypeOrgAdmin,
	)
	if err != nil {
		return "", fmt.Errorf("models.CreateOrgV1: failed to execute insert statement for org_users: %w", err)
	}

	return orgUuid, nil
}
