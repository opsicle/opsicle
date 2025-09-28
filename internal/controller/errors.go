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
	ErrorInvalidTemplate         = errors.New("invalid_template")
	ErrorInvalidVerificationCode = errors.New("invalid_verification_code")
	ErrorInvitationExists        = errors.New("invitation_exists")
	ErrorInvitationInvalid       = errors.New("invitation_invalid")
	ErrorInsufficientPermissions = errors.New("insufficient_permissions")
	ErrorLastManagerOfResource   = errors.New("last_manager_of_resource")
	ErrorLastUserInResource      = errors.New("last_user_in_resource")
	ErrorMfaRequired             = errors.New("mfa_required")
	ErrorMfaTokenInvalid         = errors.New("mfa_token_invalid")
	ErrorNotFound                = errors.New("not_found")
	ErrorOrgExists               = errors.New("org_exists")
	ErrorOrgRequiresOneAdmin     = errors.New("org_requires_one_admin")
	ErrorUserExistsInOrg         = errors.New("user_exists_in_org")
	ErrorSessionExpired          = errors.New("session_expired")
	ErrorTotpInvalid             = errors.New("totp_invalid")
	ErrorUnrecognisedMfaType     = errors.New("unknown_mfa_type")
	ErrorUnknown                 = errors.New("unknown_error")

	ErrorCodeIssue     = errors.New("code_issue")
	ErrorDatabaseIssue = errors.New("database_issue")
	ErrorQueueIssue    = errors.New("queue_issue")
)
