package controller

import "errors"

var (
	ErrorAccountSuspended        = errors.New("account_suspended")
	ErrorAuthRequired            = errors.New("auth_required")
	ErrorEmailExists             = errors.New("email_exists")
	ErrorEmailUnverified         = errors.New("email_unverified")
	ErrorGeneric                 = errors.New("generic_error")
	ErrorInvalidCredentials      = errors.New("invalid_credentials")
	ErrorInvalidInput            = errors.New("invalid_input")
	ErrorInvalidVerificationCode = errors.New("invalid_verification_code")
	ErrorMfaRequired             = errors.New("mfa_required")
	ErrorMfaTokenInvalid         = errors.New("mfa_token_invalid")
	ErrorSessionExpired          = errors.New("session_expired")
	ErrorTotpInvalid             = errors.New("totp_invalid")
	ErrorUnrecognisedMfaType     = errors.New("unknown_mfa_type")
	ErrorUnknown                 = errors.New("unknown_error")

	ErrorDatabaseIssue = errors.New("__database_issue")
	ErrorCodeIssue     = errors.New("__code_issue")
)
