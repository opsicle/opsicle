package user

import (
	"database/sql"
	"fmt"
	"opsicle/internal/auth"

	"github.com/google/uuid"
)

type Type string

const (
	TypeSysAdmin Type = "sysadmin"
)

type CreateV1Opts struct {
	Db *sql.DB

	Email    string
	Password string
	Type     Type
}

func CreateV1(opts CreateV1Opts) error {
	userUuid := uuid.New().String()
	passwordHash, err := auth.HashPassword(opts.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %s", err)
	}

	stmt, err := opts.Db.Prepare(`
	INSERT INTO users(
		id,
		email,
		password_hash,
		type
	) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %s", err)
	}

	res, err := stmt.Exec(userUuid, opts.Email, passwordHash, "sysadmin")
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
