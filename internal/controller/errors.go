package controller

import "errors"

var (
	ErrorInvalidPublicServerUrl    = errors.New("invalid_public_server_url")
	ErrorMissingApiKeys            = errors.New("missing_api_keys")
	ErrorMissingCacheConnection    = errors.New("missing_cache_connection")
	ErrorMissingDatabaseConnection = errors.New("missing_db_connection")
	ErrorMissingEmailConfig        = errors.New("missing_email_config")
	ErrorMissingQueueConnection    = errors.New("missing_queue_connection")
	ErrorMissingServiceLog         = errors.New("missing_service_log")
)
