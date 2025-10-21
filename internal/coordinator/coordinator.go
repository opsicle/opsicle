package coordinator

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/queue"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type NewOpts struct {
	Context         context.Context
	GrpcAddr        string
	GrpcCert        tls.Certificate
	GrpcCa          []byte
	HttpAddr        string
	LivenessChecks  healthcheckProbes
	ReadinessChecks healthcheckProbes
	Services        Services

	ServiceLogs chan<- common.ServiceLog
}

func New(opts NewOpts) (*coordinatorInstance, error) {
	grpcEndpoint := opts.GrpcAddr
	listener, err := net.Listen("tcp", grpcEndpoint)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			return nil, fmt.Errorf("gRPC endpoint %s already in use", grpcEndpoint)
		}
		return nil, fmt.Errorf("failed to check gRPC endpoint availability: %w", err)
	}
	if closeErr := listener.Close(); closeErr != nil {
		return nil, fmt.Errorf("failed to release gRPC endpoint after availability check: %w", closeErr)
	}

	httpEndpoint := opts.HttpAddr
	httpListener, err := net.Listen("tcp", httpEndpoint)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			return nil, fmt.Errorf("HTTP endpoint %s already in use", httpEndpoint)
		}
		return nil, fmt.Errorf("failed to check HTTP endpoint availability: %w", err)
	}
	if closeErr := httpListener.Close(); closeErr != nil {
		return nil, fmt.Errorf("failed to release HTTP endpoint after availability check: %w", closeErr)
	}

	instance := &coordinatorInstance{
		Context: opts.Context,
		HttpServer: HttpServer{
			Addr:            opts.HttpAddr,
			LivenessChecks:  opts.LivenessChecks,
			ReadinessChecks: opts.ReadinessChecks,
			ServiceLogs:     opts.ServiceLogs,
		},
		GrpcServer: GrpcServer{
			Addr:        opts.GrpcAddr,
			Cert:        opts.GrpcCert,
			CaPem:       opts.GrpcCa,
			ServiceLogs: opts.ServiceLogs,
		},
		Services: opts.Services,
	}

	if opts.ServiceLogs != nil {
		instance.serviceLogs = &opts.ServiceLogs
	} else {
		initNoopServiceLog()
		var serviceLogs chan<- common.ServiceLog = noopServiceLog
		instance.serviceLogs = &serviceLogs
		go startNoopServiceLog()
		defer stopNoopServiceLog()
	}

	return instance, nil
}

type coordinatorInstance struct {
	Context    context.Context
	HttpServer HttpServer
	GrpcServer GrpcServer
	Services   Services

	serviceLogs *chan<- common.ServiceLog
}

type Services struct {
	Queue queue.Queue
}

// Start starts the coordinator application
func (c *coordinatorInstance) Start() error {
	*c.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "starting http server for coordinator...")
	if c.HttpServer.Errors == nil {
		c.HttpServer.Errors = make(chan error, 1)
	}
	c.HttpServer.Listen()

	*c.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "starting grpc server for coordinator...")
	if c.GrpcServer.Errors == nil {
		c.GrpcServer.Errors = make(chan error, 1)
	}
	c.GrpcServer.Listen()

	if c.Context != nil {
		httpErrs := c.HttpServer.Errors
		grpcErrs := c.GrpcServer.Errors
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

		var shutdownOnce sync.Once
		shutdown := func() {
			if c.HttpServer.Instance != nil {
				if err := c.HttpServer.Instance.Shutdown(context.Background()); err != nil && !errors.Is(err, http.ErrServerClosed) {
					panic(fmt.Errorf("failed to shutdown http server: %w", err))
				}
			}
			if c.GrpcServer.Instance != nil {
				c.GrpcServer.Instance.GracefulStop()
			}
		}

		go func() {
			defer signal.Stop(signalCh)
			for {
				select {
				case sig := <-signalCh:
					if sig != nil {
						*c.serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "received signal %s, shutting down coordinator...", sig.String())
					} else {
						*c.serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "received shutdown signal, shutting down coordinator...")
					}
					shutdownOnce.Do(shutdown)
					return
				case err, ok := <-httpErrs:
					if !ok {
						httpErrs = nil
						continue
					}
					if err != nil {
						*c.serviceLogs <- common.ServiceLogf(common.LogLevelError, "http server error: %v", err)
						shutdownOnce.Do(shutdown)
						return
					}
				case err, ok := <-grpcErrs:
					if !ok {
						grpcErrs = nil
						continue
					}
					if err != nil {
						*c.serviceLogs <- common.ServiceLogf(common.LogLevelError, "grpc server error: %v", err)
						shutdownOnce.Do(shutdown)
						return
					}
				case <-c.Context.Done():
					shutdownOnce.Do(shutdown)
					return
				}
			}
		}()
	}

	httpDone := c.HttpServer.Done
	grpcDone := c.GrpcServer.Done

	if httpDone != nil {
		<-httpDone
	}
	if grpcDone != nil {
		<-grpcDone
	}

	return nil
}
