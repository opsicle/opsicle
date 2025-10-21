package coordinator

import (
	"errors"
	"fmt"
	"net/http"
	"opsicle/internal/common"

	"github.com/gorilla/mux"
)

type HttpServer struct {
	Addr            string
	Done            chan struct{}
	Errors          chan error
	Instance        *http.Server
	LivenessChecks  healthcheckProbes
	ReadinessChecks healthcheckProbes

	ServiceLogs chan<- common.ServiceLog
}

func (h *HttpServer) Listen() {
	httpAddr := h.Addr
	h.Done = make(chan struct{})
	h.Instance = &http.Server{Addr: httpAddr}

	handler := mux.NewRouter()
	registerHealthcheckRoutes(
		RouteRegistrationOpts{
			Router:      handler,
			ServiceLogs: h.ServiceLogs,
		},
		HealthcheckOpts{
			LivenessChecks:  h.LivenessChecks,
			ReadinessChecks: h.ReadinessChecks,
		},
	)
	registerMetricsRoutes(RouteRegistrationOpts{
		Router:      handler,
		ServiceLogs: h.ServiceLogs,
	})
	h.Instance.Handler = handler
	done := h.Done
	go func() {
		defer close(done)
		h.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "starting http listener...")
		if serveErr := h.Instance.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			h.Errors <- fmt.Errorf("failed to start http server: %w", serveErr)
		}
	}()
}
