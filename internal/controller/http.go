package controller

import (
	"database/sql"
	"net/http"
	"opsicle/internal/common"

	"opsicle/internal/controller/docs"
	_ "opsicle/internal/controller/docs"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/swaggo/swag"
)

type HttpApplicationOpts struct {
	AdminToken         string
	DatabaseConnection *sql.DB
	ServiceLogs        chan<- common.ServiceLog
}

// GetHttpApplication godoc
// @title           Opsicle Controller Service
// @version         1.0
// @description     API for Opsicle Controller
// @contact.name		API Support
// @contact.email		support@opsicle.io
// @tags 					  controller-service
// @host            localhost:54321
// @BasePath        /
//
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description			Used for authenticating with endpoints
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

	swag.Register(docs.SwaggerInfo.InstanceName(), docs.SwaggerInfo)
	handler.PathPrefix("/docs").Handler(httpSwagger.WrapHandler)

	return handler
}
