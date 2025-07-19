package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
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

func GetBasicAuthMiddleware(serviceLogs chan<- ServiceLog, username, password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, p, ok := r.BasicAuth()
			if !ok || u != username || p != password {
				w.Header().Set("WWW-Authenticate", `Basic realm="restricted"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func GetBearerAuthMiddleware(serviceLogs chan<- ServiceLog, expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != expectedToken {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func GetIpAllowlistMiddleware(serviceLogs chan<- ServiceLog, allowedCidrs []*net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ipAddress, err := extractRequestIp(r)
			if err != nil {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			if !isIpAllowed(ipAddress, allowedCidrs) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ParseCidrs parses and validates CIDRs
func ParseCidrs(cidrs []string) (validCidrs []*net.IPNet, warnings []string, err error) {
	var parsed []*net.IPNet
	for _, cidr := range cidrs {
		if !strings.Contains(cidr, "/") {
			cidr = cidr + "/32"
		}
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("provided cidr[%s] is invalid, it was skipped", cidr))
		}
		parsed = append(parsed, network)
	}
	return parsed, warnings, nil
}

// extractRequestIp extracts IP from X-Forwarded-For or RemoteAddr
func extractRequestIp(r *http.Request) (net.IP, error) {
	forwardedForHeader := r.Header.Get("X-Forwarded-For")
	if forwardedForHeader != "" {
		parts := strings.Split(forwardedForHeader, ",")
		if len(parts) > 0 {
			remoteIp := strings.TrimSpace(parts[0])
			parsed := net.ParseIP(remoteIp)
			if parsed != nil {
				return parsed, nil
			}
		}
	}
	remoteIp, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, err
	}
	parsed := net.ParseIP(remoteIp)
	if parsed == nil {
		return nil, errors.New("invalid remote ip")
	}
	return parsed, nil
}

// isIpAllowed checks if the IP is inside any of the allowed CIDRs
func isIpAllowed(ip net.IP, cidrs []*net.IPNet) bool {
	for _, cidr := range cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}
