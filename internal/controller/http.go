package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"opsicle/internal/cache"
	"opsicle/internal/common"
	"opsicle/internal/persistence"
	"opsicle/internal/queue"
	"strings"

	"opsicle/internal/controller/docs"
	_ "opsicle/internal/controller/docs"
	"opsicle/internal/controller/models"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/swaggo/swag"
)

type HttpApplicationOpts struct {
	ApiKeys             []string
	CacheConnection     *persistence.Redis
	DatabaseConnection  *persistence.Mysql
	EmailConfig         *SmtpServerConfig
	LivenessChecks      []func() error
	ReadinessChecks     []func() error
	PublicServerUrl     string
	QueueConnection     *persistence.Nats
	ServiceLogs         chan<- common.ServiceLog
	SessionSigningToken string
}

func (o HttpApplicationOpts) Validate() error {
	errs := []error{}

	if o.ApiKeys == nil {
		errs = append(errs, fmt.Errorf("failed to receive api key: %w", ErrorMissingApiKeys))
	}

	if o.CacheConnection == nil {
		errs = append(errs, fmt.Errorf("failed to receive a cache connection: %w", ErrorMissingDatabaseConnection))
	}

	if o.DatabaseConnection == nil {
		errs = append(errs, fmt.Errorf("failed to receive a database connection: %w", ErrorMissingDatabaseConnection))
	}

	if o.EmailConfig == nil {
		errs = append(errs, fmt.Errorf("failed to receive email configuration: %w", ErrorMissingEmailConfig))
	}

	if o.QueueConnection == nil {
		errs = append(errs, fmt.Errorf("failed to receive a queue connection: %w", ErrorMissingQueueConnection))
	}

	if o.ServiceLogs == nil {
		errs = append(errs, fmt.Errorf("failed to receive a service log: %w", ErrorMissingServiceLog))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
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
func GetHttpApplication(opts HttpApplicationOpts) (http.Handler, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to initialise http application: %w", err)
	}

	// initialise common global

	serviceLogs = &opts.ServiceLogs

	apiKeys = opts.ApiKeys
	*serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "controller has %v api keys registered", apiKeys)

	dbInstance = opts.DatabaseConnection.GetClient()

	cache.InitRedis(cache.InitRedisOpts{
		RedisConnection: opts.CacheConnection,
		ServiceLogs:     *serviceLogs,
	})
	cacheInstance = cache.Get()

	queue.InitNats(queue.InitNatsOpts{
		NatsConnection: opts.QueueConnection,
		ServiceLogs:    *serviceLogs,
	})
	queueInstance = queue.Get()

	var err error
	publicServerUrl, err = url.Parse(opts.PublicServerUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server url '%s': %w: %w", opts.PublicServerUrl, ErrorInvalidPublicServerUrl, err)
	}

	if opts.SessionSigningToken != "" {
		models.SetSessionSigningToken(opts.SessionSigningToken)
	}

	if opts.EmailConfig == nil {
		*serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "email is not enabled")
	} else {
		smtpConfig = *opts.EmailConfig
		if err := smtpConfig.VerifyConnection(); err != nil {
			*serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to authenticate with the provided smtp configuration: %s", err)
			smtpConfig = SmtpServerConfig{}
		}
	}
	*serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "email status: %v", smtpConfig.IsSet())

	handler := mux.NewRouter()
	handler.NotFoundHandler = common.GetNotFoundHandler()
	common.RegisterCommonHttpEndpoints(common.CommonHttpEndpointsOpts{
		Router:          handler,
		ServiceLogs:     *serviceLogs,
		LivenessChecks:  opts.LivenessChecks,
		ReadinessChecks: opts.ReadinessChecks,
	})

	api := handler.PathPrefix("/api").Subrouter()
	apiOpts := RouteRegistrationOpts{
		Router:      api,
		ServiceLogs: *serviceLogs,
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
		*serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "registered route[%s] with methods[%s]", pathTemplate, strings.Join(methods, "|"))
		return nil
	}); err != nil {
		return nil, err
	}

	return handler, nil
}
