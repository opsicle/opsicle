package controller

import "fmt"

var (
	ErrorAuthRequired       = fmt.Errorf("auth_required")
	ErrorGeneric            = fmt.Errorf("generic_error")
	ErrorInvalidCredentials = fmt.Errorf("invalid_credentials")
	ErrorInvalidEndpoint    = fmt.Errorf("invalid_endpoint")
	ErrorMfaRequired        = fmt.Errorf("mfa_required")
	ErrorMfaTokenInvalid    = fmt.Errorf("mfa_token_invalid")
	ErrorUserLoginFailed    = fmt.Errorf("login_failed")
	ErrorEmailUnverified    = fmt.Errorf("email_unverified")

	ErrorOutputNil        = fmt.Errorf("__output_nil")
	ErrorOutputNotPointer = fmt.Errorf("__output_not_pointer")

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
)
