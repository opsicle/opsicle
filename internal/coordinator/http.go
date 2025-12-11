package coordinator

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"opsicle/internal/cache"
	"opsicle/internal/common"
	"opsicle/internal/persistence"
	"opsicle/internal/queue"

	"github.com/gorilla/mux"
)

type HttpApplicationOpts struct {
	Cache            *persistence.Redis
	ControllerApiKey string
	ControllerUrl    string
	Queue            *persistence.Nats

	LivenessChecks  []func() error
	ReadinessChecks []func() error

	CacheConnection *persistence.Redis
	QueueConnection *persistence.Nats

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

	// initialise common global

	serviceLogs = &opts.ServiceLogs
	*serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "controller has %v api key(s) registered", len(apiKeys))

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
	controllerUrl, err = url.Parse(opts.ControllerUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse controllerUrl[%s]: %w", opts.ControllerUrl, err)
	}

	handler := mux.NewRouter()

	api := handler.PathPrefix("/api").Subrouter()
	apiOpts := RouteRegistrationOpts{
		Router:      api,
		ServiceLogs: opts.ServiceLogs,
	}

	registerInitRoutes(apiOpts)
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
