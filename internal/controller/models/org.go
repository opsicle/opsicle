package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
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
}

type AddUserToOrgV1 struct {
	Db *sql.DB

	UserId string
}

func (o *Org) AddUserV1(opts AddUserToOrgV1) error {
	sqlStmt := `
	INSERT INTO org_users(
		org_id,
		user_id
	) VALUES (?, ?)`
	sqlArgs := []any{*o.Id, opts.UserId}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return fmt.Errorf("org.Org.AddUserV1: failed to prepare insert statement: %s", err)
	}

	res, err := stmt.Exec(sqlArgs...)
	if err != nil {
		return fmt.Errorf("org.Org.AddUserV1: failed to execute insert statement: %s", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("org.Org.AddUserV1: failed to retrieve the number of rows affected: %s", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("org.Org.AddUserV1: failed to insert only 1 user")
	}
	return nil
}

type GetOrgUserV1Opts struct {
	Db *sql.DB

	UserId string
}

func (o *Org) GetUserV1(opts GetOrgUserV1Opts) (*User, error) {
	sqlStmt := `
		SELECT
			users.email,
			users.id,
			orgs.id,
			orgs.code,
			org_users.joined_at
			FROM org_users
				JOIN orgs ON orgs.id = org_users.org_id
				JOIN users ON users.id = org_users.user_id
			WHERE
				org_id = ? AND user_id = ?
	`
	sqlArgs := []any{*o.Id, opts.UserId}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("models.Org.GetUserV1: failed to prepare select statement: %s", err)
	}

	res := stmt.QueryRow(sqlArgs...)
	if res.Err() != nil {
		return nil, fmt.Errorf("models.Org.GetUserV1: failed to execute select statement: %s", err)
	}
	var userInstance User
	userInstance.Org = &Org{}
	if err := res.Scan(
		&userInstance.Email,
		&userInstance.Id,
		&userInstance.Org.Id,
		&userInstance.Org.Code,
		&userInstance.JoinedOrgAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("models.Org.GetUserV1: failed to load selected data into memory: %s", err)
	}

	return &userInstance, nil
}

type LoadOrgUserCountV1Opts struct {
	Db *sql.DB
}

func (o *Org) LoadUserCountV1(opts LoadOrgUserCountV1Opts) (int, error) {
	sqlStmt := `
		SELECT COUNT(*) AS 'count'
			FROM org_users
			WHERE
				org_id = ?
	`
	sqlArgs := []any{*o.Id}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return -1, fmt.Errorf("models.Org.GetUserCountV1: failed to prepare select statement: %s", err)
	}

	res := stmt.QueryRow(sqlArgs...)
	if res.Err() != nil {
		return -1, fmt.Errorf("models.Org.GetUserCountV1: failed to execute select statement: %s", err)
	}
	if err := res.Scan(&o.UserCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, nil
		}
		return -1, fmt.Errorf("models.Org.GetUserCountV1: failed to load selected data into memory: %s", err)
	}

	return *o.UserCount, nil
}
