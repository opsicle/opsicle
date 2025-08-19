package models

import "fmt"

var (
	ErrorCredentialsAuthenticationFailed = fmt.Errorf("credentials_authentication_failed")
	ErrorDuplicateEntry                  = fmt.Errorf("duplicate_entry")
	ErrorNotFound                        = fmt.Errorf("not_found")
	ErrorUnknown                         = fmt.Errorf("unknown_error")
	ErrorUserEmailNotVerified            = fmt.Errorf("email_not_verified")
	ErrorUserDisabled                    = fmt.Errorf("user_disabled")
	ErrorUserDeleted                     = fmt.Errorf("user_deleted")

	errorNoDatabaseConnection  = fmt.Errorf("no_database_connection")
	errorInputValidationFailed = fmt.Errorf("input_validation_failed")

	mysqlErrorDuplicateEntryCode uint16 = 1062
)
