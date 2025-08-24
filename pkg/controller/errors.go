package controller

import (
	"errors"
)

var (
	ErrorAuthRequired       = errors.New("auth_required")
	ErrorGeneric            = errors.New("generic_error")
	ErrorEmailExists        = errors.New("email_exists")
	ErrorEmailUnverified    = errors.New("email_unverified")
	ErrorHealthcheckFailed  = errors.New("healthcheck_failed")
	ErrorInvalidCredentials = errors.New("invalid_credentials")
	ErrorInvalidEndpoint    = errors.New("invalid_endpoint")
	ErrorInvitationExists   = errors.New("invitation_exists")
	ErrorMfaRequired        = errors.New("mfa_required")
	ErrorMfaTokenInvalid    = errors.New("mfa_token_invalid")
	ErrorOrgExists          = errors.New("org_exists")
	ErrorSessionExpired     = errors.New("session_expired")
	ErrorUserExistsInOrg    = errors.New("user_exists_in_org")
	ErrorUserLoginFailed    = errors.New("login_failed")

	// ErrorJwtTokenExpired indicates the token has expired
	ErrorJwtTokenExpired = errors.New("jwt_token_expired")
	// ErrorJwtTokenSignature indicates token signature validation failed
	ErrorJwtTokenSignature = errors.New("jwt_token_signature")
	// ErrorJwtClaims indicates that the claim data couldn't be parsed
	ErrorJwtClaimsInvalid = errors.New("jwt_claims_invalid")

	ErrorOutputNil        = errors.New("__output_nil")
	ErrorOutputNotPointer = errors.New("__output_not_pointer")

	ErrorConnectionRefused  = errors.New("__connection_refused")
	ErrorConnectionTimedOut = errors.New("__connection_timed_out")

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

	ErrorInvalidInput = errors.New("unexpected end of JSON input")

	ErrorUnknown = errors.New("unknown_error")
)
