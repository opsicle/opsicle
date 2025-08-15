package controller

import "fmt"

var (
	ErrorAuthRequired         = fmt.Errorf("auth_required")
	ErrorUserLoginFailed      = fmt.Errorf("login_failed")
	ErrorUserEmailNotVerified = fmt.Errorf("email_not_verified")
	ErrorGeneric              = fmt.Errorf("generic_error")
)
