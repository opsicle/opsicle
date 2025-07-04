package approver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	handler.HandleFunc("/approval", func(w http.ResponseWriter, r *http.Request) {
		log := r.Context().Value("logger").(requestLogger)
		keys, err := Cache.Scan(approvalRequestCachePrefix)
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to retrieve approval requests: %s", err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to retrieve approvals",
				Success: false,
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
			return
		}

		if len(keys) == 0 {
			res, _ := json.Marshal(httpResponse{
				Message: "no approval requests found",
				Success: false,
			})
			w.WriteHeader(http.StatusNotFound)
			w.Write(res)
			return
		}

		res, _ := json.Marshal(httpResponse{
			Data:    keys,
			Success: true,
		})
		w.WriteHeader(http.StatusNotFound)
		w.Write(res)

	}).Methods(http.MethodGet)

	handler.HandleFunc("/approval", func(w http.ResponseWriter, r *http.Request) {
		var req ApprovalRequest
		log := r.Context().Value("logger").(requestLogger)

		log(common.LogLevelDebug, "reading request body...")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to read request body: %s", err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to read request body",
				Success: false,
			})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}

		log(common.LogLevelDebug, "parsing request body...")
		err = json.Unmarshal(body, &req)
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to parse request body: %s", err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to parse request body",
				Success: false,
			})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}

		log(common.LogLevelDebug, fmt.Sprintf("storing approvalRequest[%s]...", req.Id))
		if err = Cache.Set(approvalRequestCachePrefix+req.Id, "0", 0); err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to store approvalRequest[%s]: %s", req.Id, err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to storing approval request",
				Success: false,
			})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}

		for _, target := range req.Telegram {
			log(common.LogLevelDebug, fmt.Sprintf("sending approval to chat[%v]...", target.ChatId))
			err = TelegramApprover.SendApproval(req)
			if err != nil {
				log(common.LogLevelError, fmt.Sprintf("failed to send approval request message[%v]: %s", target.ChatId, err))
				res, _ := json.Marshal(httpResponse{
					Message: "failed to send approval request message to telegram",
					Success: false,
				})
				w.WriteHeader(http.StatusBadRequest)
				w.Write(res)
				return
			}
		}

		res, _ := json.Marshal(httpResponse{
			Message: "ok",
			Success: true,
		})
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	}).Methods(http.MethodPost)

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
				serviceLogs <- common.ServiceLogf(level, "req[%s] %s", requestId, message)
			}))
			serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "req[%s] received %s at %s", requestId, r.Method, r.RequestURI)
			next.ServeHTTP(w, r.WithContext(requestContext))
			serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "req[%s] completed in %v", requestId, time.Since(start))
		})
	}
}
