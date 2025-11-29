package approver

import (
	"fmt"
	"opsicle/internal/common"

	"opsicle/internal/approver/docs"
	_ "opsicle/internal/approver/docs"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/swaggo/swag"
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
// @tags 					  approver-service
// @host            localhost:12345
// @BasePath        /
func StartHttpServer(opts StartHttpServerOpts) error {
	router := mux.NewRouter()

	registerHealthcheckRoutes(RouteRegistrationOpts{
		Router:      router,
		ServiceLogs: opts.ServiceLogs,
	})

	for urlPath, routeHandlers := range routesMapping {
		for method, getRouteHandler := range routeHandlers {
			router.HandleFunc(urlPath, getRouteHandler()).Methods(method)
		}
	}

	swag.Register(docs.SwaggerInfo.InstanceName(), docs.SwaggerInfo)
	router.PathPrefix("/docs").Handler(httpSwagger.WrapHandler)

	router.NotFoundHandler = common.GetNotFoundHandler()

	serverOpts := common.NewHttpServerOpts{
		Addr:        opts.Addr,
		Done:        opts.Done,
		Handler:     router,
		ServiceLogs: opts.ServiceLogs,
	}

	if opts.BasicAuth != nil {
		serverOpts.BasicAuth = &common.NewHttpServerBasicAuthOpts{
			Username: opts.BasicAuth.Username,
			Password: opts.BasicAuth.Password,
		}
	}

	if opts.BearerAuth != nil {
		serverOpts.BearerAuth = &common.NewHttpServerBearerAuthOpts{
			Token: opts.BearerAuth.Token,
		}
	}

	if opts.IpAllowlist != nil {
		serverOpts.IpAllowlist = &common.NewHttpServerIpAllowlistOpts{
			AllowedIps: opts.IpAllowlist.AllowedIps,
		}
	}

	server, err := common.NewHttpServer(serverOpts)
	if err != nil {
		return fmt.Errorf("failed to create a http server: %w", err)
	}
	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}
