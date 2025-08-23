package controller

import (
	"errors"
	"fmt"
)

var (
	ErrorAuthRequired       = fmt.Errorf("auth_required")
	ErrorGeneric            = fmt.Errorf("generic_error")
	ErrorEmailExists        = errors.New("email_exists")
	ErrorInvalidCredentials = fmt.Errorf("invalid_credentials")
	ErrorInvalidEndpoint    = fmt.Errorf("invalid_endpoint")
	ErrorMfaRequired        = fmt.Errorf("mfa_required")
	ErrorMfaTokenInvalid    = fmt.Errorf("mfa_token_invalid")
	ErrorOrgExists          = errors.New("org_exists")
	ErrorSessionExpired     = fmt.Errorf("session_expired")
	ErrorUserLoginFailed    = fmt.Errorf("login_failed")
	ErrorEmailUnverified    = fmt.Errorf("email_unverified")

	// ErrorJwtTokenExpired indicates the token has expired
	ErrorJwtTokenExpired = errors.New("jwt_token_expired")
	// ErrorJwtTokenSignature indicates token signature validation failed
	ErrorJwtTokenSignature = errors.New("jwt_token_signature")
	// ErrorJwtClaims indicates that the claim data couldn't be parsed
	ErrorJwtClaimsInvalid = errors.New("jwt_claims_invalid")

	ErrorOutputNil        = fmt.Errorf("__output_nil")
	ErrorOutputNotPointer = fmt.Errorf("__output_not_pointer")

	ErrorConnectionRefused  = fmt.Errorf("__connection_refused")
	ErrorConnectionTimedOut = fmt.Errorf("__connection_timed_out")

	ErrorClientMarshalInput              = fmt.Errorf("__client_marshal_input")
	ErrorClientRequestCreation           = fmt.Errorf("__client_request_creation")
	ErrorClientRequestExecution          = fmt.Errorf("__client_request_execution")
	ErrorClientResponseNotFromController = fmt.Errorf("__client_response_not_from_controller")
	ErrorClientResponseReading           = fmt.Errorf("__client_response_reading")
	ErrorClientUnmarshalResponse         = fmt.Errorf("__client_unmarshal_response")
	ErrorClientMarshalResponseData       = fmt.Errorf("__client_marshal_response_data")
	ErrorClientUnmarshalOutput           = fmt.Errorf("__client_unmarshal_output")
	ErrorClientUnmarshalErrorCode        = fmt.Errorf("__client_unmarshal_error_code")
	ErrorClientUnsuccessfulResponse      = fmt.Errorf("__client_unsuccessful_response")

	ErrorInvalidInput = fmt.Errorf("unexpected end of JSON input")

	ErrorUnknown = errors.New("unknown_error")
)
