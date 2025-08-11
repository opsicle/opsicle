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
	errorCode ...error,
) {
	log := request.Context().Value(HttpContextLogger).(HttpRequestLogger)
	log(LogLevelError, message)
	responseData := HttpResponse{
		Message: message,
		Success: false,
	}
	if len(errorCode) > 0 {
		responseData.Data = errorCode[0].Error()
	} else {
		responseData.Data = "generic_error"
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
