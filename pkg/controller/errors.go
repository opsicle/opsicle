package controller

import "fmt"

var (
	ErrorAuthRequired         = fmt.Errorf("auth_required")
	ErrorUserLoginFailed      = fmt.Errorf("login_failed")
	ErrorUserEmailNotVerified = fmt.Errorf("email_not_verified")
	ErrorMfaRequired          = fmt.Errorf("mfa_required")
	ErrorGeneric              = fmt.Errorf("generic_error")
)
