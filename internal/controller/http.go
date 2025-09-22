package controller

import (
	"database/sql"
	"net/http"
	"net/url"
	"opsicle/internal/common"
	"strings"

	"opsicle/internal/controller/docs"
	_ "opsicle/internal/controller/docs"
	"opsicle/internal/controller/models"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/swaggo/swag"
)

type HttpApplicationOpts struct {
	AdminToken          string
	DatabaseConnection  *sql.DB
	EmailConfig         *SmtpServerConfig
	LivenessChecks      []func() error
	ReadinessChecks     []func() error
	PublicServerUrl     *url.URL
	ServiceLogs         chan<- common.ServiceLog
	SessionSigningToken string
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

	if opts.SessionSigningToken != "" {
		models.SetSessionSigningToken(opts.SessionSigningToken)
	}

	if opts.EmailConfig == nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelWarn, "email is not enabled")
	} else {
		smtpConfig = *opts.EmailConfig
		if err := smtpConfig.VerifyConnection(); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to authenticate with the provided smtp configuration: %s", err)
			smtpConfig = SmtpServerConfig{}
		}
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "email status: %v", smtpConfig.IsSet())

	if publicServerUrl == "" {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelWarn, "the public server url has not been set, some urls issued might not be accurate")
	}

	if serviceLogs == nil {
		noopServiceLogs := make(chan common.ServiceLog, 32)
		go func() {
			if _, ok := <-noopServiceLogs; !ok {
				logrus.Infof("what")
				return
			}
		}()
		var logsReceiver chan<- common.ServiceLog = noopServiceLogs
		SetServiceLogs(&logsReceiver)
	} else {
		SetServiceLogs(&opts.ServiceLogs)
	}

	handler := mux.NewRouter()
	handler.NotFoundHandler = common.GetNotFoundHandler()
	if opts.LivenessChecks != nil {
		livenessChecks = append(livenessChecks, opts.LivenessChecks...)
	}
	if opts.ReadinessChecks != nil {
		readinessChecks = append(readinessChecks, opts.ReadinessChecks...)
	}
	registerHealthcheckRoutes(RouteRegistrationOpts{
		Router:      handler,
		ServiceLogs: opts.ServiceLogs,
	})
	handler.Handle("/metrics", promhttp.Handler())
	if opts.AdminToken != "" {
		admin := handler.PathPrefix("/admin").Subrouter()
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

	registerAutomationRoutes(apiOpts)
	registerAutomationTemplatesRoutes(apiOpts)
	registerOrgRoutes(apiOpts)
	registerSessionRoutes(apiOpts)
	registerUserRoutes(apiOpts)

	swag.Register(docs.SwaggerInfo.InstanceName(), docs.SwaggerInfo)
	handler.PathPrefix("/docs").Handler(httpSwagger.WrapHandler)

	if err := handler.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		// Get path template
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			return nil
		}
		methods, err := route.GetMethods()
		if err != nil {
			methods = []string{"*"}
		}
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "registered route[%s] with methods[%s]", pathTemplate, strings.Join(methods, "|"))
		return nil
	}); err != nil {
		return nil
	}

	return handler
}
