package models

import (
	"database/sql"
	"fmt"
	"opsicle/internal/validate"
	"time"

	"github.com/google/uuid"
)

type User struct {
	Id                    *string    `json:"id"`
	Email                 string     `json:"email"`
	IsEmailVerified       bool       `json:"isEmailVerfieid"`
	EmailVerificationCode string     `json:"emailVerificationCode"`
	EmailVerifiedAt       *time.Time `json:"emailVerifiedAt"`

	// Password doesn't exist in the database
	Password *string `json:"password"`

	// PasswordHash is the hash of the password loaded from the database
	PasswordHash *string `json:"passwordHash"`

	// Type is the type of the user according to the system
	Type UserType `json:"type"`

	// CreatedAt is the timestamp when the user was created
	CreatedAt time.Time `json:"createdAt"`

	// IsDeleted will be truthy if a user is deleted
	IsDeleted bool `json:"isDeleted"`

	// DeletedAt is the timestamp of a user's deletion if IsDeleted
	// is truthy
	DeletedAt *time.Time `json:"deletedAt"`

	// IsDisabled will be truthy if a user is disabled
	IsDisabled bool `json:"isDisabled"`

	// DisabledAt is the timestamp of a user's disabling if IsDisabled
	// is truthy
	DisabledAt *time.Time `json:"disabledAt"`

	// LastUpdatedAt is the timestamp when the user row was last updated,
	// possibly `nil` if the user was just created
	LastUpdatedAt *time.Time `json:"lastUpdatedAt"`

	// Mfas is not populated on `Load*`
	Mfas []UserMfa `json:"mfa"`

	// Orgs is not populated on `Load*``
	Orgs []Org `json:"orgs"`
}

func (u User) GetId() string {
	return *u.Id
}

func (u User) GetRedacted() User {
	return User{
		Id:              u.Id,
		Email:           u.Email,
		IsEmailVerified: u.IsEmailVerified,
		EmailVerifiedAt: u.EmailVerifiedAt,
		Type:            u.Type,
		CreatedAt:       u.CreatedAt,
		LastUpdatedAt:   u.LastUpdatedAt,
		IsDeleted:       u.IsDeleted,
		DeletedAt:       u.DeletedAt,
		IsDisabled:      u.IsDisabled,
		DisabledAt:      u.DisabledAt,
	}
}

func (u User) IsVerified() bool {
	return u.EmailVerificationCode == ""
}

func (u *User) LoadByEmailV1(opts DatabaseConnection) error {
	if u.Email == "" {
		return fmt.Errorf("missing email")
	} else if err := validate.Email(u.Email); err != nil {
		return fmt.Errorf("invalid email")
	}
	return u.load(opts, SelectorUserEmail)
}

func (u *User) LoadByIdV1(opts DatabaseConnection) error {
	if u.Id == nil {
		return fmt.Errorf("missing id")
	} else if _, err := uuid.Parse(*u.Id); err != nil {
		return fmt.Errorf("invalid id")
	}
	return u.load(opts, SelectorUserId)
}

func (u *User) LoadByVerificationCodeV1(opts DatabaseConnection) error {
	if u.EmailVerificationCode == "" {
		return fmt.Errorf("missing verification code")
	}
	return u.load(opts, SelectorUserVerificationCode)
}

func (u *User) load(opts DatabaseConnection, selector UserLoadSelector) error {
	var selectorField string
	selectorArgs := []any{}
	switch selector {
	case SelectorUserId:
		selectorField = "id"
		selectorArgs = append(selectorArgs, *u.Id)
	case SelectorUserEmail:
		selectorField = "email"
		selectorArgs = append(selectorArgs, u.Email)
	case SelectorUserVerificationCode:
		selectorField = "email_verification_code"
		selectorArgs = append(selectorArgs, u.EmailVerificationCode)
	}
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(`
			SELECT
				id,
				email,
				is_email_verified,
				email_verification_code,
				email_verified_at,
				password_hash,
				type,
				created_at,
				last_updated_at,
				is_deleted,
				deleted_at,
				is_disabled,
				disabled_at
				FROM users
					WHERE %s = ?
			`,
			selectorField,
		),
		Args:     selectorArgs,
		FnSource: fmt.Sprintf("models.User.load[%s]", selectorField),
		ProcessRow: func(r *sql.Row) error {
			return r.Scan(
				&u.Id,
				&u.Email,
				&u.IsEmailVerified,
				&u.EmailVerificationCode,
				&u.EmailVerifiedAt,
				&u.PasswordHash,
				&u.Type,
				&u.CreatedAt,
				&u.LastUpdatedAt,
				&u.IsDeleted,
				&u.DeletedAt,
				&u.IsDisabled,
				&u.DisabledAt,
			)
		},
	}); err != nil {
		return err
	}
	return nil
}
