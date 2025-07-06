package approver

import (
	"context"
	"fmt"
	"net/http"
	"opsicle/internal/common"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type StartHttpServerOpts struct {
	Addr        string
	Done        chan common.Done
	ServiceLogs chan<- common.ServiceLog
}

func StartHttpServer(opts StartHttpServerOpts) error {
	handler := mux.NewRouter()

	handler.Use(getRequestLoggerMiddleware(opts.ServiceLogs))

	handler.HandleFunc("/approval-request", getListApprovalRequestsHandler()).Methods(http.MethodGet)
	handler.HandleFunc("/approval-request/{requestId}/{requestUuid}", getGetApprovalRequestHandler()).Methods(http.MethodGet)
	handler.HandleFunc("/approval/{approvalId}", getGetApprovalHandler()).Methods(http.MethodGet)
	handler.HandleFunc("/approval-request", getCreateApprovalRequestHandler()).Methods(http.MethodPost)

	server := http.Server{
		Addr:              opts.Addr,
		Handler:           handler,
		IdleTimeout:       common.DefaultDurationConnectionTimeout,
		ReadTimeout:       common.DefaultDurationConnectionTimeout,
		ReadHeaderTimeout: common.DefaultDurationConnectionTimeout,
		WriteTimeout:      common.DefaultDurationConnectionTimeout,
	}

	logrus.Infof("Starting HTTP server on %s...", opts.Addr)
	go func() {
		<-opts.Done
		if err := server.Close(); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "server closed: %s", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start server: %s", err)
	}
	return nil
}

func getRequestLoggerMiddleware(serviceLogs chan<- common.ServiceLog) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestId := uuid.New().String()
			requestContext := context.WithValue(r.Context(), "requestId", requestId)
			requestContext = context.WithValue(requestContext, "logger", requestLogger(func(level string, message string) {
				serviceLogs <- common.ServiceLogf(level, "req[%s] %s", requestId, message)
			}))
			serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "req[%s] received %s at %s", requestId, r.Method, r.RequestURI)
			next.ServeHTTP(w, r.WithContext(requestContext))
			serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "req[%s] completed in %v", requestId, time.Since(start))
		})
	}
}
