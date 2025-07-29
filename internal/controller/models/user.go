package models

import (
	"database/sql"
	"fmt"
	"opsicle/internal/auth"
	"time"
)

type AddUserToOrgV1Opts struct {
	Db *sql.DB

	OrgId string
}

type User struct {
	Id           *string    `json:"id"`
	Email        string     `json:"email"`
	Password     *string    `json:"password"`
	PasswordHash *string    `json:"passwordHash"`
	Org          *Org       `json:"org"`
	JoinedOrgAt  *time.Time `json:"joinedOrgAt"`
	Type         UserType   `json:"type"`
}

func (u User) AddToOrgV1(opts AddUserToOrgV1Opts) error {
	stmt, err := opts.Db.Prepare(`
	INSERT INTO org_users(
		user_id,
		org_id
	) VALUES (?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %s", err)
	}

	res, err := stmt.Exec(u.Id, opts.OrgId)
	if err != nil {
		return fmt.Errorf("failed to execute insert statement: %s", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve the number of rows affected: %s", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("failed to insert only 1 user")
	}
	return nil
}

func (u User) ValidatePassword() bool {
	return auth.ValidatePassword(*u.Password, *u.PasswordHash)
}
