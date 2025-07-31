package common

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type HttpResponse struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

func GetNotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		SendHttpFailResponse(w, r, http.StatusNotFound, "not found", fmt.Errorf("endpoint[%s] not found", r.URL.Path))
	}
}

func SendHttpFailResponse(
	responseWriter http.ResponseWriter,
	request *http.Request,
	statusCode int,
	message string,
	errorDetails error,
	data ...any,
) {
	log := request.Context().Value(HttpContextLogger).(HttpRequestLogger)
	log(LogLevelError, fmt.Sprintf("%s: %s", message, errorDetails))
	responseData := HttpResponse{
		Message: message,
		Success: false,
	}
	if len(data) > 0 {
		responseData.Data = data[0]
	}
	res, _ := json.Marshal(responseData)
	responseWriter.WriteHeader(statusCode)
	responseWriter.Write(res)
}

func SendHttpSuccessResponse(
	responseWriter http.ResponseWriter,
	request *http.Request,
	statusCode int,
	message string,
	data ...any,
) {
	responseData := HttpResponse{
		Message: message,
		Success: true,
	}
	if len(data) > 0 {
		responseData.Data = data[0]
	}
	res, _ := json.Marshal(responseData)
	responseWriter.WriteHeader(statusCode)
	responseWriter.Write(res)
}
