package cli

import "errors"

var (
	ErrorAuthError             = errors.New("auth_error")
	ErrorControllerUnavailable = errors.New("controller_unavailable")
	ErrorClientUnavailable     = errors.New("client_unavailable")
	ErrorInvalidInput          = errors.New("invalid_input")
	ErrorNotAuthenticated      = errors.New("not_authenticated")
)
