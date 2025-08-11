package controller

import "fmt"

var (
	ErrorAuthRequired         = fmt.Errorf("auth required")
	ErrorUserLoginFailed      = fmt.Errorf("login failed")
	ErrorUserEmailNotVerified = fmt.Errorf("email not verified")
)
