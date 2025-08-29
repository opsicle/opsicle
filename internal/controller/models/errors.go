package models

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrorCredentialsAuthenticationFailed = errors.New("credentials_authentication_failed")
	ErrorDuplicateEntry                  = errors.New("duplicate_entry")
	ErrorGenericDatabaseIssue            = errors.New("generic_database_issue")
	ErrorInsertFailed                    = errors.New("insert_failed")
	ErrorNotFound                        = errors.New("not_found")
	ErrorQueryFailed                     = errors.New("query_failed")
	ErrorRowsAffectedCheckFailed         = errors.New("rows_affected_check_failed")
	ErrorSelectFailed                    = errors.New("select_failed")
	ErrorStmtPreparationFailed           = errors.New("stmt_preparation_failed")
	ErrorUnknown                         = errors.New("unknown_error")
	ErrorUserDeleted                     = errors.New("user_deleted")
	ErrorUserDisabled                    = errors.New("user_disabled")
	ErrorUserEmailNotVerified            = errors.New("email_not_verified")

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
