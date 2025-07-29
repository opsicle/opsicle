package models

import (
	"database/sql"
	"errors"
	"fmt"
	"opsicle/internal/auth"

	"github.com/google/uuid"
)

type CreateUserV1Opts struct {
	Db *sql.DB

	OrgCode  string
	Email    string
	Password string
	Type     UserType
}

func (o CreateUserV1Opts) Validate() error {
	errs := []error{}

	if o.Db == nil {
		errs = append(errs, fmt.Errorf("no database connection supplied"))
	}
	if o.OrgCode == "" {
		errs = append(errs, fmt.Errorf("no org code supplied"))
	}
	if o.Email == "" {
		errs = append(errs, fmt.Errorf("no email supplied"))
	}
	if o.Password == "" {
		errs = append(errs, fmt.Errorf("no password supplied"))
	}
	if o.Type == "" {
		errs = append(errs, fmt.Errorf("no user type supplied"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func CreateUserV1(opts CreateUserV1Opts) error {
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("models.CreateUserV1: failed to validate input arguments: %s", err)
	}
	userUuid := uuid.New().String()
	passwordHash, err := auth.HashPassword(opts.Password)
	if err != nil {
		return fmt.Errorf("models.CreateUserV1: failed to hash password: %s", err)
	}
	userType := opts.Type

	orgQueryStmt, err := opts.Db.Prepare(`
	SELECT id FROM orgs WHERE code = ?`)
	if err != nil {
		return fmt.Errorf("models.CreateUserV1: failed to prepare select statement for org[%s]: %s", opts.OrgCode, err)
	}
	row := orgQueryStmt.QueryRow(opts.OrgCode)
	if row.Err() != nil {
		return fmt.Errorf("models.CreateUserV1: failed to query org[%s]: %s", opts.OrgCode, err)
	}
	var orgId string
	if err := row.Scan(&orgId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("models.CreateUserV1: org[%s] does not exist", opts.OrgCode)
		}
		return fmt.Errorf("models.CreateUserV1: failed to scan org results: %s", err)
	}

	userInsertStmt, err := opts.Db.Prepare(`INSERT INTO users(id, email, password_hash, type) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("models.CreateUserV1: failed to prepare insert statement to create user: %s", err)
	}
	if _, err := userInsertStmt.Exec(userUuid, opts.Email, passwordHash, userType); err != nil {
		return fmt.Errorf("models.CreateUserV1: failed to execute insert statement to create user: %s", err)
	}

	orgUserInsertStmt, err := opts.Db.Prepare(`
	INSERT INTO org_users(user_id, org_id) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("models.CreateUserV1: failed to prepare insert statement: %s", err)
	}

	if _, err := orgUserInsertStmt.Exec(userUuid, orgId); err != nil {
		return fmt.Errorf("models.CreateUserV1: failed to execute insert statement to add user to organisation: %s", err)
	}

	return nil
}
