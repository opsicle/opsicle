package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Org struct {
	Id         *string    `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  *time.Time `json:"createdAt"`
	UpdatedAt  *time.Time `json:"updatedAt"`
	IsDeleted  bool       `json:"isDeleted"`
	DeletedAt  *time.Time `json:"deletedAt"`
	IsDisabled bool       `json:"isDisabled"`
	DisabledAt *time.Time `json:"disabledAt"`
	Code       string     `json:"code"`
	Icon       *string    `json:"icon"`
	Logo       *string    `json:"logo"`
	Motd       *string    `json:"motd"`
	Type       string     `json:"type"`
	UserCount  *int       `json:"userCount"`
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
