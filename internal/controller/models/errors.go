package models

import "fmt"

var (
	ErrorCredentialsAuthenticationFailed = fmt.Errorf("credentials authentication failed")
	ErrorUserEmailNotVerified            = fmt.Errorf("email not verified")
	ErrorUserDisabled                    = fmt.Errorf("user disabled")
	ErrorUserDeleted                     = fmt.Errorf("user deleted")
)
