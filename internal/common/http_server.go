package common

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type HttpServer struct {
	Done        chan Done
	Server      http.Server
	ServiceLogs chan<- ServiceLog
}

func (s *HttpServer) Shutdown() error {
	s.ServiceLogs <- ServiceLogf(LogLevelInfo, "shutting down http server at %s...", s.Server.Addr)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.Server.Shutdown(ctx); err != nil {
		return fmt.Errorf("http server closed: %s", err)
	}
	return nil
}

func (s *HttpServer) Start() error {
	s.ServiceLogs <- ServiceLogf(LogLevelInfo, "starting http server on %s...", s.Server.Addr)
	if err := s.Server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("failed to start http server: %w", err)
		}
	}
	return nil
}

type NewHttpServerOpts struct {
	// Addr specifies the interface address which the server should
	// listen on
	Addr string

	// BasicAuth when defined, sets the server up so that BasicAuth
	// is required
	BasicAuth *NewHttpServerBasicAuthOpts

	// BearerAuth when defined, sets the server up so that token-based
	// authentication is required
	BearerAuth *NewHttpServerBearerAuthOpts

	// Done is the channel through which a message will come through to
	// let the server know to initiate a graceful shutdown
	Done chan Done

	// IpAllowlist when defined, sets the server up so that only
	// connections from the addresses defined here are allowed to
	// connect to the server
	IpAllowlist *NewHttpServerIpAllowlistOpts

	// Handler is the
	Handler http.Handler

	// ServiceLogs is where logs are sent to
	ServiceLogs chan<- ServiceLog
}

type NewHttpServerBasicAuthOpts struct {
	Username string
	Password string
}

type NewHttpServerBearerAuthOpts struct {
	Token string
}

type NewHttpServerIpAllowlistOpts struct {
	AllowedIps []string
}

func NewHttpServer(opts NewHttpServerOpts) (*HttpServer, error) {
	logger := GetRequestLoggerMiddleware(opts.ServiceLogs)
	metrics := GetCommonMetricsMiddleware(opts.ServiceLogs)

	// var router = mux.NewRouter()
	// router.NotFoundHandler = GetNotFoundHandler()

	var handler http.Handler = opts.Handler

	if opts.BasicAuth != nil {
		if opts.BasicAuth.Username == "" || opts.BasicAuth.Password == "" {
			return nil, fmt.Errorf("failed to receive a set of valid credentials for basic auth")
		}
		if len(opts.BasicAuth.Password) < 8 {
			opts.ServiceLogs <- ServiceLogf(LogLevelWarn, "provided basic auth password is less than 8 characters and maybe weak/brute-forceable in a reasonable amount of time")
		}
		basicAuther := GetBasicAuthMiddleware(opts.ServiceLogs, opts.BasicAuth.Username, opts.BasicAuth.Password)
		handler = basicAuther(handler)
	}

	if opts.BearerAuth != nil {
		if opts.BearerAuth.Token == "" {
			return nil, fmt.Errorf("failed to receive a set of valid credentials for bearer auth")
		}
		if len(opts.BearerAuth.Token) < 16 {
			opts.ServiceLogs <- ServiceLogf(LogLevelWarn, "provided bearer token is less than 16 characters and maybe weak/brute-forceable in a reasonable amount of time")
		}
		bearerAuther := GetBearerAuthMiddleware(opts.ServiceLogs, opts.BearerAuth.Token)
		handler = bearerAuther(handler)
	}

	if opts.IpAllowlist != nil {
		cidrs, warnings, err := ParseCidrs(opts.IpAllowlist.AllowedIps)
		if err != nil {
			return nil, fmt.Errorf("failed to parse provided cidrs['%s']: %s", strings.Join(opts.IpAllowlist.AllowedIps, "', '"), err)
		}
		for warningIndex, warning := range warnings {
			opts.ServiceLogs <- ServiceLogf(LogLevelWarn, "received warning[%v] while parsing cidrs: %s", warningIndex, warning)
		}
		ipAllowLister := GetIpAllowlistMiddleware(opts.ServiceLogs, cidrs)
		handler = ipAllowLister(handler)
	}

	handler = logger(metrics(handler))

	return &HttpServer{
		Done: opts.Done,
		Server: http.Server{
			Addr:              opts.Addr,
			Handler:           handler,
			IdleTimeout:       DefaultDurationConnectionTimeout,
			ReadTimeout:       DefaultDurationConnectionTimeout,
			ReadHeaderTimeout: DefaultDurationConnectionTimeout,
			WriteTimeout:      DefaultDurationConnectionTimeout,
		},
		ServiceLogs: opts.ServiceLogs,
	}, nil
}
