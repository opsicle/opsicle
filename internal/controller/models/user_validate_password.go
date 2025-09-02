package models

import "opsicle/internal/auth"

func (u User) ValidatePassword(input string) bool {
	return auth.ValidatePassword(input, *u.PasswordHash)
}
