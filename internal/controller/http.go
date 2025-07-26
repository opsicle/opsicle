package controller

import (
	"database/sql"
	"net/http"
	"opsicle/internal/common"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HttpApplicationOpts struct {
	AdminToken         string
	DatabaseConnection *sql.DB
	ServiceLogs        chan<- common.ServiceLog
}

func GetHttpApplication(opts HttpApplicationOpts) http.Handler {
	db = opts.DatabaseConnection

	handler := mux.NewRouter()
	handler.NotFoundHandler = common.GetNotFoundHandler()

	handler.Handle("/metrics", promhttp.Handler())
	admin := handler.PathPrefix("/admin").Subrouter()

	if opts.AdminToken != "" {
		registerAdminRoutes(RouteRegistrationOpts{
			Router:      admin,
			ServiceLogs: opts.ServiceLogs,
		}, opts.AdminToken)
	}

	api := handler.PathPrefix("/api").Subrouter()
	apiOpts := RouteRegistrationOpts{
		Router:      api,
		ServiceLogs: opts.ServiceLogs,
	}

	registerAutomationTemplatesRoutes(apiOpts)
	registerSessionRoutes(apiOpts)

	return handler
}
