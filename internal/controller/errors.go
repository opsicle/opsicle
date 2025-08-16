package controller

import "errors"

var (
	ErrorAuthRequired        = errors.New("auth_required")
	ErrorInvalidPasword      = errors.New("invalid_password")
	ErrorUnrecognisedMfaType = errors.New("unknown_mfa_type")
	ErrorMfaRequired         = errors.New("mfa_required")
	ErrorGeneric             = errors.New("generic_error")
)
