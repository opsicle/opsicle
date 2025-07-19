package approver

import (
	"fmt"
	"net/http"
	"opsicle/internal/common"
	"strings"

	_ "opsicle/internal/docs"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

type StartHttpServerOpts struct {
	Addr        string
	BasicAuth   *StartHttpServerBasicAuthOpts
	BearerAuth  *StartHttpServerBearerAuthOpts
	Done        chan common.Done
	IpAllowlist *StartHttpServerIpAllowlistOpts
	ServiceLogs chan<- common.ServiceLog
}

type StartHttpServerBasicAuthOpts struct {
	Username string
	Password string
}

type StartHttpServerBearerAuthOpts struct {
	Token string
}

type StartHttpServerIpAllowlistOpts struct {
	AllowedIps []string
}

// StartHttpServer godoc
// @title           Opsicle Approver API
// @version         1.0
// @description     API for Opsicle Approver service
// @host            localhost:12345
// @BasePath        /
func StartHttpServer(opts StartHttpServerOpts) error {
	router := mux.NewRouter()

	for urlPath, routeHandlers := range routesMapping {
		for method, getRouteHandler := range routeHandlers {
			router.HandleFunc(urlPath, getRouteHandler()).Methods(method)
		}
	}

	router.PathPrefix("/docs").Handler(httpSwagger.WrapHandler)

	router.NotFoundHandler = getNotFoundHandler()

	logger := common.GetRequestLoggerMiddleware(opts.ServiceLogs)

	var handler http.Handler = router

	if opts.BasicAuth != nil {
		if opts.BasicAuth.Username == "" || opts.BasicAuth.Password == "" {
			return fmt.Errorf("failed to receive a set of valid credentials for basic auth")
		}
		if len(opts.BasicAuth.Password) < 8 {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelWarn, "provided basic auth password is less than 8 characters and maybe weak/brute-forceable in a reasonable amount of time")
		}
		basicAuther := common.GetBasicAuthMiddleware(opts.ServiceLogs, opts.BasicAuth.Username, opts.BasicAuth.Password)
		handler = basicAuther(handler)
	}

	if opts.BearerAuth != nil {
		if opts.BearerAuth.Token == "" {
			return fmt.Errorf("failed to receive a set of valid credentials for bearer auth")
		}
		if len(opts.BearerAuth.Token) < 16 {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelWarn, "provided bearer token is less than 16 characters and maybe weak/brute-forceable in a reasonable amount of time")
		}
		bearerAuther := common.GetBasicAuthMiddleware(opts.ServiceLogs, opts.BasicAuth.Username, opts.BasicAuth.Password)
		handler = bearerAuther(handler)
	}

	if opts.IpAllowlist != nil {
		cidrs, warnings, err := common.ParseCidrs(opts.IpAllowlist.AllowedIps)
		if err != nil {
			return fmt.Errorf("failed to parse provided cidrs['%s']: %s", strings.Join(opts.IpAllowlist.AllowedIps, "', '"), err)
		}
		for warningIndex, warning := range warnings {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelWarn, "received warning[%v] while parsing cidrs: %s", warningIndex, warning)
		}
		ipAllowLister := common.GetIpAllowlistMiddleware(opts.ServiceLogs, cidrs)
		handler = ipAllowLister(handler)
	}

	handler = logger(handler)

	server := http.Server{
		Addr:              opts.Addr,
		Handler:           handler,
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
