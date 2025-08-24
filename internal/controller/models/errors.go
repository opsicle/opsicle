package models

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrorCredentialsAuthenticationFailed = errors.New("credentials_authentication_failed")
	ErrorDuplicateEntry                  = errors.New("duplicate_entry")
	ErrorNotFound                        = errors.New("not_found")
	ErrorUnknown                         = errors.New("unknown_error")
	ErrorUserEmailNotVerified            = errors.New("email_not_verified")
	ErrorUserDisabled                    = errors.New("user_disabled")
	ErrorUserDeleted                     = errors.New("user_deleted")

	ErrorInvalidInput = errors.New("invalid_input")

	errorNoDatabaseConnection  = errors.New("no_database_connection")
	errorInputValidationFailed = errors.New("input_validation_failed")

	mysqlErrorDuplicateEntryCode uint16 = 1062
)

func isMysqlDuplicateError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		if mysqlErr.Number == mysqlErrorDuplicateEntryCode {
			return true
		}
	}
	return false
}
