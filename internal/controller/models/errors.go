package models

import (
	"errors"
)

var (
	ErrorCredentialsAuthenticationFailed = errors.New("credentials_authentication_failed")
	ErrorDatabaseUndefined               = errors.New("database_undefined")
	ErrorDeleteFailed                    = errors.New("delete_failed")
	ErrorDuplicateEntry                  = errors.New("duplicate_entry")
	ErrorGenericDatabaseIssue            = errors.New("generic_database_issue")
	ErrorIdRequired                      = errors.New("id_required")
	ErrorInsertFailed                    = errors.New("insert_failed")
	ErrorNotFound                        = errors.New("not_found")
	ErrorQueryFailed                     = errors.New("query_failed")
	ErrorRowsAffectedCheckFailed         = errors.New("rows_affected_check_failed")
	ErrorSelectFailed                    = errors.New("select_failed")
	ErrorSelectsFailed                   = errors.New("selects_failed")
	ErrorStmtPreparationFailed           = errors.New("stmt_preparation_failed")
	ErrorUpdateFailed                    = errors.New("update_failed")
	ErrorUnknown                         = errors.New("unknown_error")
	ErrorUserDeleted                     = errors.New("user_deleted")
	ErrorUserDisabled                    = errors.New("user_disabled")
	ErrorUserEmailNotVerified            = errors.New("email_not_verified")
	ErrorVersionRequired                 = errors.New("version_required")

	ErrorInvalidInput = errors.New("invalid_input")

	errorNoDatabaseConnection  = errors.New("no_database_connection")
	errorInputValidationFailed = errors.New("input_validation_failed")

	mysqlErrorDuplicateEntryCode uint16 = 1062
)
