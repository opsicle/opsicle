package models

import (
	"database/sql"
	"errors"
	"fmt"
	"opsicle/internal/auth"
	"opsicle/internal/common"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

type CreateUserV1Opts struct {
	Db *sql.DB

	OrgCode  *string
	Email    string
	Password string
	Type     UserType
}

func (o CreateUserV1Opts) Validate() error {
	errs := []error{}

	if o.Db == nil {
		errs = append(errs, fmt.Errorf("no database connection supplied"))
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
		return fmt.Errorf("models.CreateUserV1: failed to validate input arguments: %w", err)
	}
	userUuid := uuid.New().String()
	passwordHash, err := auth.HashPassword(opts.Password)
	if err != nil {
		return fmt.Errorf("models.CreateUserV1: failed to hash password: %w", err)
	}
	userType := opts.Type
	emailVerificationCode, err := common.GenerateRandomString(32)
	if err != nil {
		return fmt.Errorf("models.CreateUserV1: failed to generate a random string: %w", err)
	}

	userInsertStmt, err := opts.Db.Prepare(`
		INSERT INTO users(
			id,
			email,
			email_verification_code,
			password_hash,
			type
		) VALUES (?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("models.CreateUserV1: failed to prepare insert statement to create user: %w", err)
	}
	if _, err := userInsertStmt.Exec(
		userUuid,
		opts.Email,
		emailVerificationCode,
		passwordHash,
		userType,
	); err != nil {
		return fmt.Errorf("models.CreateUserV1: failed to execute insert statement to create user: %w", err)
	}

	if opts.OrgCode != nil {
		var orgId string
		orgQueryStmt, err := opts.Db.Prepare(`SELECT id FROM orgs WHERE code = ?`)
		if err != nil {
			return fmt.Errorf("models.CreateUserV1: failed to prepare select statement for org[%s]: %s", *opts.OrgCode, err)
		}
		row := orgQueryStmt.QueryRow(opts.OrgCode)
		if row.Err() != nil {
			return fmt.Errorf("models.CreateUserV1: failed to query org[%s]: %s", *opts.OrgCode, err)
		}
		if err := row.Scan(&orgId); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("models.CreateUserV1: org[%s] does not exist", *opts.OrgCode)
			}
			return fmt.Errorf("models.CreateUserV1: failed to scan org results: %w", err)
		}
		orgUserInsertStmt, err := opts.Db.Prepare(`
		INSERT INTO org_users(user_id, org_id) VALUES (?, ?)`)
		if err != nil {
			return fmt.Errorf("models.CreateUserV1: failed to prepare insert statement: %w", err)
		}

		if _, err := orgUserInsertStmt.Exec(userUuid, orgId); err != nil {
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) {
				if mysqlErr.Number == mysqlErrorDuplicateEntryCode {
					return fmt.Errorf("models.CreateUserV1: failed to insert a duplicate user: %w", ErrorDuplicateEntry)
				}
			}
			return fmt.Errorf("models.CreateUserV1: failed to execute insert statement to add user to organisation: %w", err)
		}
	}

	return nil
}
