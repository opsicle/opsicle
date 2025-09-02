package models

import (
	"database/sql"
	"errors"
	"fmt"
	"opsicle/internal/auth"
	"opsicle/internal/common"

	"github.com/google/uuid"
)

type CreateUserV1Opts struct {
	Db *sql.DB

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

	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			INSERT INTO users(
				id,
				email,
				email_verification_code,
				password_hash,
				type
			) VALUES (?, ?, ?, ?, ?)
			`,
		Args: []any{
			userUuid,
			opts.Email,
			emailVerificationCode,
			passwordHash,
			userType,
		},
		FnSource:     "models.CreateUserV1",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return err
	}

	return nil
}
