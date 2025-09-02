package models

import (
	"database/sql"

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

	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			INSERT INTO orgs(
				id,
				name,
				code,
				type,
				created_by
			) VALUES (?, ?, ?, ?, ?)
		`,
		Args: []any{
			orgUuid,
			opts.Name,
			opts.Code,
			string(opts.Type),
			opts.UserId,
		},
		FnSource:     "models.CreateOrgV1",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return "", err
	}

	org := Org{Id: &orgUuid}
	if err := org.AddUserV1(AddUserToOrgV1{
		Db:         opts.Db,
		UserId:     opts.UserId,
		MemberType: string(TypeOrgAdmin),
	}); err != nil {
		return "", err
	}

	return orgUuid, nil
}
