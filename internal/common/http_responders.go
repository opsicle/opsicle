package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrorEndpointHandlerNotFound = errors.New("endpoint_handler_not_found")
)

type HttpResponse struct {
	Data    any    `json:"data"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

func GetNotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		SendHttpFailResponse(w, r, http.StatusNotFound, fmt.Sprintf("handler for request[%s %s] not found", r.Method, r.URL.Path), ErrorEndpointHandlerNotFound)
	}
}

func SendHttpFailResponse(
	responseWriter http.ResponseWriter,
	request *http.Request,
	statusCode int,
	message string,
	errorCode error,
	data ...any,
) {
	log := request.Context().Value(HttpContextLogger).(HttpRequestLogger)
	log(LogLevelError, message)
	responseData := HttpResponse{
		Code:    errorCode.Error(),
		Message: message,
		Success: false,
	}
	if len(data) > 0 {
		responseData.Data = data[0]
	}
	res, _ := json.Marshal(responseData)
	responseWriter.WriteHeader(statusCode)
	select {
	case <-request.Context().Done():
		log(LogLevelError, fmt.Sprintf("client[%s] disconnected before response, aborting sending of response", request.RemoteAddr))
		return
	default:
	}
	byteCount, err := responseWriter.Write(res)
	if err != nil {
		log(LogLevelError, fmt.Sprintf("failed to write response: %s", err))
		return
	}
	log(LogLevelTrace, fmt.Sprintf("responded with %v bytes", byteCount))
}

func SendHttpSuccessResponse(
	responseWriter http.ResponseWriter,
	request *http.Request,
	statusCode int,
	message string,
	data ...any,
) {
	log := request.Context().Value(HttpContextLogger).(HttpRequestLogger)
	responseData := HttpResponse{
		Code:    "success",
		Message: message,
		Success: true,
	}
	if len(data) > 0 {
		responseData.Data = data[0]
	}
	res, _ := json.Marshal(responseData)
	responseWriter.WriteHeader(statusCode)
	select {
	case <-request.Context().Done():
		log(LogLevelError, fmt.Sprintf("client[%s] disconnected before response, aborting sending of response", request.RemoteAddr))
		return
	default:
	}
	byteCount, err := responseWriter.Write(res)
	if err != nil {
		log(LogLevelError, fmt.Sprintf("failed to write response: %s", err))
		return
	}
	log(LogLevelTrace, fmt.Sprintf("responded with %v bytes", byteCount))
}
