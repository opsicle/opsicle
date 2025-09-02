package models

import (
	"database/sql"
	"fmt"
	"opsicle/internal/auth"

	"github.com/google/uuid"
)

type UpdateUserPasswordV1Input struct {
	Db *sql.DB

	NewPassword string
}

func (u *User) UpdatePasswordV1(opts UpdateUserPasswordV1Input) error {
	if u.Id == nil {
		return fmt.Errorf("missing id")
	} else if _, err := uuid.Parse(*u.Id); err != nil {
		return fmt.Errorf("invalid id")
	}
	passwordHash, err := auth.HashPassword(opts.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	return u.UpdateFieldsV1(UpdateUserFieldsV1{
		Db: opts.Db,
		FieldsToSet: map[string]any{
			"password_hash": passwordHash,
		},
	})
}
