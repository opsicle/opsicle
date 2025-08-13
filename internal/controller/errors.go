package controller

import "errors"

var (
	ErrorAuthRequired        = errors.New("auth_required")
	ErrorInvalidPasword      = errors.New("invalid_password")
	ErrorUnrecognisedMfaType = errors.New("unknown_mfa_type")
)
