package models

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type CreateUserMfaV1Opts struct {
	Db *sql.DB

	Secret *string
	Config *string
	UserId string
	Type   string
}

func CreateUserMfaV1(opts CreateUserMfaV1Opts) (*UserMfa, error) {
	mfaId := uuid.NewString()
	insertFields := []string{
		"id",
		"user_id",
		"type",
	}
	var insertValues []string
	sqlArgs := []any{
		mfaId,
		opts.UserId,
		opts.Type,
	}

	if opts.Secret != nil {
		insertFields = append(insertFields, "secret")
		sqlArgs = append(sqlArgs, *opts.Secret)
	}
	if opts.Config != nil {
		insertFields = append(insertFields, "config_json")
		sqlArgs = append(sqlArgs, *opts.Config)
	}

	insertValues = make([]string, len(insertFields))
	for i := 0; i < len(insertValues); i++ {
		insertValues[i] = "?"
	}

	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(
			"INSERT INTO user_mfa(%s) VALUES (%s)",
			strings.Join(insertFields, ", "),
			strings.Join(insertValues, ", "),
		),
		Args:         sqlArgs,
		FnSource:     "models.CreateUserMfaV1",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
	}

	return &UserMfa{
		Id:     mfaId,
		Secret: opts.Secret,
		UserId: opts.UserId,
		Type:   opts.Type,
	}, nil
}
