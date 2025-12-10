package coordinator

import "errors"

var (
	ErrorMissingCache            = errors.New("missing_cache")
	ErrorMissingControllerApiKey = errors.New("missing_controller_api_key")
	ErrorMissingQueue            = errors.New("missing_queue")
)
