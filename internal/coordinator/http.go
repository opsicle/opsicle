package coordinator

import (
	"errors"
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/persistence"

	"github.com/gorilla/mux"
)

type HttpApplicationOpts struct {
	Cache            *persistence.Redis
	ControllerApiKey string
	Queue            *persistence.Nats

	LivenessChecks  []func() error
	ReadinessChecks []func() error

	ServiceLogs chan<- common.ServiceLog
}

func (o HttpApplicationOpts) Validate() error {
	var errs = []error{}

	if o.Cache == nil {
		errs = append(errs, ErrorMissingCache)
	}

	if o.ControllerApiKey == "" {
		errs = append(errs, ErrorMissingControllerApiKey)
	}

	if o.Queue == nil {
		errs = append(errs, ErrorMissingQueue)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func GetHttpApplication(opts HttpApplicationOpts) (http.Handler, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

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
	return handler, nil
}
