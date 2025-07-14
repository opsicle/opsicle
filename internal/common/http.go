package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type HttpContextKey string

const (
	HttpContextRequestId HttpContextKey = "http-request-id"
	HttpContextLogger    HttpContextKey = "http-logger"
)

type HttpRequestLogger func(string, string)

type HttpResponse struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

func AddHttpHeaders(req *http.Request) {
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-App-Id", "opsicle")
}

func NewHttpClient() *http.Client {
	return &http.Client{
		Timeout: DefaultDurationConnectionTimeout,
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
		responseData.Data = data
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

func GetRequestLoggerMiddleware(serviceLogs chan<- ServiceLog) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			var requestId string
			if r.Header.Get("X-Trace-Id") != "" {
				requestId = r.Header.Get("X-Trace-Id")
			} else {
				requestId = uuid.New().String()
			}
			requestContext := context.WithValue(r.Context(), HttpContextRequestId, requestId)
			requestContext = context.WithValue(requestContext, HttpContextLogger, HttpRequestLogger(func(level string, message string) {
				serviceLogs <- ServiceLogf(level, "req[%s] %s", requestId, message)
			}))
			serviceLogs <- ServiceLogf(LogLevelDebug, "req[%s] received %s at %s", requestId, r.Method, r.RequestURI)
			next.ServeHTTP(w, r.WithContext(requestContext))
			serviceLogs <- ServiceLogf(LogLevelInfo, "req[%s] [%s %s %s %s] from remote[%s] completed in %v", requestId, r.Proto, r.Host, r.Method, r.RequestURI, r.RemoteAddr, time.Since(start))
		})
	}
}
