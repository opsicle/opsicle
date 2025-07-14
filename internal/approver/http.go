package approver

import (
	"fmt"
	"net/http"
	"opsicle/internal/common"

	"github.com/gorilla/mux"
)

type StartHttpServerOpts struct {
	Addr        string
	Done        chan common.Done
	ServiceLogs chan<- common.ServiceLog
}

func StartHttpServer(opts StartHttpServerOpts) error {
	handler := mux.NewRouter()

	for urlPath, routeHandlers := range routesMapping {
		for method, getRouteHandler := range routeHandlers {
			handler.HandleFunc(urlPath, getRouteHandler()).Methods(method)
		}
	}

	handler.NotFoundHandler = getNotFoundHandler()

	logger := common.GetRequestLoggerMiddleware(opts.ServiceLogs)

	loggedHandler := logger(handler)

	server := http.Server{
		Addr:              opts.Addr,
		Handler:           loggedHandler,
		IdleTimeout:       common.DefaultDurationConnectionTimeout,
		ReadTimeout:       common.DefaultDurationConnectionTimeout,
		ReadHeaderTimeout: common.DefaultDurationConnectionTimeout,
		WriteTimeout:      common.DefaultDurationConnectionTimeout,
	}

	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "Starting HTTP server on %s...", opts.Addr)
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
