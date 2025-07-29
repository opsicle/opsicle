package org

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type OrgUser struct {
	UserId   string
	OrgId    string
	JoinedAt time.Time
}

type GetUserV1Opts struct {
	Db *sql.DB

	UserId string
}

func (o *Org) GetUserV1(opts GetUserV1Opts) (*OrgUser, error) {
	stmt, err := opts.Db.Prepare(`
	SELECT
		user_id,
		org_id,
		joined_at
		FROM org_users
		WHERE
			org_id = ?
			AND user_id = ?`)
	if err != nil {
		return nil, fmt.Errorf("org.Org.GetUserV1: failed to prepare select statement: %s", err)
	}

	res := stmt.QueryRow(*o.Id, opts.UserId)
	if res.Err() != nil {
		return nil, fmt.Errorf("org.Org.GetUserV1: failed to execute select statement: %s", err)
	}
	var orgUser OrgUser
	if err := res.Scan(
		&orgUser.UserId,
		&orgUser.OrgId,
		&orgUser.JoinedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("org.Org.GetUserV1: failed to load selected data into memory: %s", err)
	}

	return &orgUser, nil
}
