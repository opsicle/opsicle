package coordinator

import (
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/persistence"

	"github.com/gorilla/mux"
)

type HttpApplicationOpts struct {
	Cache *persistence.Redis
	Queue *persistence.Nats

	LivenessChecks  []func() error
	ReadinessChecks []func() error

	ServiceLogs chan<- common.ServiceLog
}

func GetHttpApplication(opts HttpApplicationOpts) http.Handler {
	handler := mux.NewRouter()

	api := handler.PathPrefix("/api").Subrouter()
	apiOpts := RouteRegistrationOpts{
		Router:      api,
		ServiceLogs: opts.ServiceLogs,
	}

	registerJobsRoutes(apiOpts)

	handler.NotFoundHandler = common.GetNotFoundHandler()
	common.RegisterCommonHttpEndpoints(common.CommonHttpEndpointsOpts{
		LivenessChecks:  opts.LivenessChecks,
		ReadinessChecks: opts.ReadinessChecks,
		Router:          handler,
		ServiceLogs:     opts.ServiceLogs,
	})
	return handler
}
