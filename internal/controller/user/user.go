package user

import "opsicle/internal/auth"

type User struct {
	Id           *string
	Email        string
	Password     *string
	PasswordHash *string
	Type         Type
}

func (u User) ValidatePassword() bool {
	return auth.ValidatePassword(*u.Password, *u.PasswordHash)
}
