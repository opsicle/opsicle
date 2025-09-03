package models

import (
	"database/sql"
)

type VerifyUserV1Opts struct {
	Db *sql.DB

	VerificationCode string
	UserAgent        string
	IpAddress        string
}

func (u *User) VerifyV1(opts VerifyUserV1Opts) error {
	u.EmailVerificationCode = opts.VerificationCode
	if err := u.LoadByVerificationCodeV1(DatabaseConnection{Db: opts.Db}); err != nil {
		return err
	}

	if err := u.UpdateFieldsV1(UpdateUserFieldsV1{
		Db: opts.Db,
		FieldsToSet: map[string]any{
			"email_verification_code":      "",
			"is_email_verified":            true,
			"email_verified_at":            DatabaseFunction("NOW()"),
			"email_verified_by_user_agent": opts.UserAgent,
			"email_verified_by_ip_address": opts.IpAddress,
		},
	}); err != nil {
		return err
	}

	return nil
}
