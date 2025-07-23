package common

import (
	"context"
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
