package approver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/config"
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

	handler.HandleFunc("/approval", func(w http.ResponseWriter, r *http.Request) {
		var req ApprovalRequest
		log := r.Context().Value("logger").(requestLogger)

		log(config.LogLevelDebug, "reading request body...")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log(config.LogLevelError, fmt.Sprintf("failed to read request body: %s", err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to read request body",
				Success: false,
			})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}

		log(config.LogLevelDebug, "parsing request body...")
		err = json.Unmarshal(body, &req)
		if err != nil {
			log(config.LogLevelError, fmt.Sprintf("failed to parse request body: %s", err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to parse request body",
				Success: false,
			})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}

		log(config.LogLevelDebug, fmt.Sprintf("storing approvalRequest[%s]...", req.Id))
		err = RedisCache.Client.Set(req.Id, string(body), 0).Err()
		if err != nil {
			log(config.LogLevelError, fmt.Sprintf("failed to store approvalRequest[%s]: %s", req.Id, err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to storing approval request",
				Success: false,
			})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}

		log(config.LogLevelDebug, fmt.Sprintf("sending approval to chat[%s]...", req.Chat))
		err = TelegramApprover.SendApproval(req, SendApprovalOpts{
			Chat: req.Chat,
		})
		if err != nil {
			log(config.LogLevelError, fmt.Sprintf("failed to send approval message[%s]: %s", req.Chat, err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to send approval message",
				Success: false,
			})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}

		res, _ := json.Marshal(httpResponse{
			Message: "ok",
			Success: true,
		})
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	}).Methods("POST")

	server := http.Server{
		Addr:              opts.Addr,
		Handler:           handler,
		IdleTimeout:       config.DefaultDurationConnectionTimeout,
		ReadTimeout:       config.DefaultDurationConnectionTimeout,
		ReadHeaderTimeout: config.DefaultDurationConnectionTimeout,
		WriteTimeout:      config.DefaultDurationConnectionTimeout,
	}

	logrus.Infof("Starting HTTP server on %s...", opts.Addr)
	go func() {
		<-opts.Done
		if err := server.Close(); err != nil {
			opts.ServiceLogs <- common.ServiceLog{config.LogLevelError, ""}
		}
	}()

	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start server: %s", err)
	}
	return nil
}

type httpResponse struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

type requestLogger func(string, string)

func getRequestLoggerMiddleware(serviceLogs chan<- common.ServiceLog) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestId := uuid.New().String()
			requestContext := context.WithValue(r.Context(), "requestId", requestId)
			requestContext = context.WithValue(requestContext, "logger", requestLogger(func(level string, message string) {
				serviceLogs <- common.ServiceLog{Level: level, Message: fmt.Sprintf("req[%s] %s", requestId, message)}
			}))
			serviceLogs <- common.ServiceLog{config.LogLevelInfo, fmt.Sprintf("req[%s] received %s at %s", requestId, r.Method, r.RequestURI)}
			next.ServeHTTP(w, r.WithContext(requestContext))
			serviceLogs <- common.ServiceLog{config.LogLevelInfo, fmt.Sprintf("req[%s] completed in %v", requestId, time.Since(start))}
		})
	}
}
