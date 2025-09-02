package models

import (
	"database/sql"
	"errors"
	"fmt"
)

type ListUserMfasV1Opts struct {
	Db *sql.DB

	// UserId when set, returns the MFA methods of a user identified
	// by their `id`. Takes precedence over the Email field
	UserId *string

	// Email when set, returns the MFA methods of a user identified
	// by their `email`
	Email *string
}

func (o ListUserMfasV1Opts) Validate() error {
	errs := []error{}
	if o.Db == nil {
		errs = append(errs, errorNoDatabaseConnection)
	}
	if o.UserId == nil && o.Email == nil {
		errs = append(errs, errorInputValidationFailed)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func ListUserMfasV1(opts ListUserMfasV1Opts) (UserMfas, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("models.ListUserMfasV1: failed to validate input: %w", err)
	}

	selectorField := ""
	selectorValue := ""
	if opts.UserId != nil {
		selectorField = "`users`.`id`"
		selectorValue = *opts.UserId
	} else if opts.Email != nil {
		selectorField = "`users`.`email`"
		selectorValue = *opts.Email
	}

	output := UserMfas{}
	if err := executeMysqlSelects(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(`
			SELECT
				user_mfa.id,
				user_mfa.type,
				user_mfa.secret,
				user_mfa.is_verified,
				user_mfa.verified_at,
				user_mfa.created_at,
				user_mfa.last_updated_at
				FROM user_mfa
					JOIN users ON users.id = user_mfa.user_id
				WHERE %s = ?
					AND user_mfa.is_verified = true
			`,
			selectorField,
		),
		Args: []any{selectorValue},
		ProcessRows: func(r *sql.Rows) error {
			userMfa := UserMfa{}
			if err := r.Scan(
				&userMfa.Id,
				&userMfa.Type,
				&userMfa.Secret,
				&userMfa.IsVerified,
				&userMfa.VerifiedAt,
				&userMfa.CreatedAt,
				&userMfa.LastUpdatedAt,
			); err != nil {
				return err
			}
			output = append(output, userMfa)
			return nil
		},
	}); err != nil {
		return nil, err
	}

	return output, nil
}
