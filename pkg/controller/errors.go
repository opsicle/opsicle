package controller

import "fmt"

var (
	ErrorUserLoginFailed      = fmt.Errorf("login failed")
	ErrorUserEmailNotVerified = fmt.Errorf("email not verified")
)
