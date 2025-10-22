package types

import "errors"

var (
	ErrorAccountSuspended        = errors.New("account_suspended")
	ErrorAuthRequired            = errors.New("auth_required")
	ErrorEmailExists             = errors.New("email_exists")
	ErrorEmailUnverified         = errors.New("email_unverified")
	ErrorGeneric                 = errors.New("generic_error")
	ErrorHealthcheckFailed       = errors.New("healthcheck_failed")
	ErrorInvalidCredentials      = errors.New("invalid_credentials")
	ErrorInvalidEndpoint         = errors.New("invalid_endpoint")
	ErrorInvalidInput            = errors.New("invalid_input")
	ErrorInvalidTemplate         = errors.New("invalid_template")
	ErrorInvalidVerificationCode = errors.New("invalid_verification_code")
	ErrorInvitationExists        = errors.New("invitation_exists")
	ErrorInvitationInvalid       = errors.New("invitation_invalid")
	ErrorInsufficientPermissions = errors.New("insufficient_permissions")
	ErrorLastManagerOfResource   = errors.New("last_manager_of_resource")
	ErrorLastOrgAdmin            = errors.New("last_org_admin")
	ErrorLastUserInResource      = errors.New("last_user_in_resource")
	ErrorMfaRequired             = errors.New("mfa_required")
	ErrorMfaTokenInvalid         = errors.New("mfa_token_invalid")
	ErrorNotFound                = errors.New("not_found")
	ErrorOrgExists               = errors.New("org_exists")
	ErrorUserExistsInOrg         = errors.New("user_exists_in_org")
	ErrorSessionExpired          = errors.New("session_expired")
	ErrorTotpInvalid             = errors.New("totp_invalid")
	ErrorUnrecognisedMfaType     = errors.New("unknown_mfa_type")
	ErrorUserLoginFailed         = errors.New("login_failed")

	ErrorClientMarshalInput              = errors.New("__client_marshal_input")
	ErrorClientRequestCreation           = errors.New("__client_request_creation")
	ErrorClientRequestExecution          = errors.New("__client_request_execution")
	ErrorClientResponseNotFromController = errors.New("__client_response_not_from_controller")
	ErrorClientResponseReading           = errors.New("__client_response_reading")
	ErrorClientUnmarshalResponse         = errors.New("__client_unmarshal_response")
	ErrorClientMarshalResponseData       = errors.New("__client_marshal_response_data")
	ErrorClientUnmarshalOutput           = errors.New("__client_unmarshal_output")
	ErrorClientUnmarshalErrorCode        = errors.New("__client_unmarshal_error_code")
	ErrorClientUnsuccessfulResponse      = errors.New("__client_unsuccessful_response")

	// ErrorJwtTokenExpired indicates the token has expired
	ErrorJwtTokenExpired = errors.New("jwt_token_expired")
	// ErrorJwtTokenSignature indicates token signature validation failed
	ErrorJwtTokenSignature = errors.New("jwt_token_signature")
	// ErrorJwtClaims indicates that the claim data couldn't be parsed
	ErrorJwtClaimsInvalid = errors.New("jwt_claims_invalid")

	ErrorCodeIssue     = errors.New("code_issue")
	ErrorDatabaseIssue = errors.New("database_issue")
	ErrorQueueIssue    = errors.New("queue_issue")
	ErrorUnknown       = errors.New("unknown_error")

	ErrorOutputNil        = errors.New("__output_nil")
	ErrorOutputNotPointer = errors.New("__output_not_pointer")

	ErrorConnectionRefused  = errors.New("__connection_refused")
	ErrorConnectionTimedOut = errors.New("__connection_timed_out")
)
