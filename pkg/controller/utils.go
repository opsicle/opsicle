package controller

import "net/http"

// isControllerResponse
func isControllerResponse(r *http.Response) bool {
	return r.StatusCode <= http.StatusInternalServerError
}

// isSuccessResponse
func isSuccessResponse(r *http.Response) bool {
	return r.StatusCode == http.StatusOK
}
