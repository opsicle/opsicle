package coordinator

import (
	"fmt"
	"net/http"
	"net/url"
	"opsicle/internal/audit"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/config"
	"opsicle/internal/coordinator"
	"opsicle/internal/persistence"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Command = cli.NewCommand(cli.CommandOpts{
	Name:  "coordinator",
	Flags: flags,
	Use:   "coordinator",
	Short: "Starts the coordinator component",
	Long:  "Starts the coordinator component which serves as the API layer that user interfaces can connect to to perform actions",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		appName := opts.GetFullname()
		serviceLogs := opts.GetServiceLogs()

		//
		// audit module
		//

		logrus.Infof("initialising audit module...")
		logrus.Debugf("connecting to mongodb...")
		mongoInstance := persistence.NewMongo(
			persistence.MongoConnectionOpts{
				AppName:  appName,
				Hosts:    viper.GetStringSlice(config.MongoHosts),
				IsDirect: true,
			},
			persistence.MongoAuthOpts{
				Password: viper.GetString(config.MongoPassword),
				Username: viper.GetString(config.MongoUsername),
			},
			&serviceLogs,
		)
		if err := mongoInstance.Init(); err != nil {
			return fmt.Errorf("failed to connect to mongo: %w", err)
		}
		logrus.Infof("connected to mongodb")
		if auditModuleError := audit.InitMongo(mongoInstance.GetClient()); auditModuleError != nil {
			return fmt.Errorf("failed to initialise audit module: %w", auditModuleError)
		}
		opts.AddShutdownProcess("mongo", mongoInstance.Shutdown)
		audit.Log(audit.LogEntry{
			EntityId:     fmt.Sprintf("%v@%s", opts.GetUserId(), opts.GetHostname()),
			EntityType:   audit.CoordinatorEntity,
			Verb:         audit.Connect,
			ResourceId:   fmt.Sprintf("%s:%v", viper.GetString("mongo-host"), viper.GetInt("mongo-port")),
			ResourceType: audit.DbResource,
		})
		logrus.Infof("initialised audit module")

		//
		// queue module
		//

		logrus.Infof("initialising queue connection...")
		logrus.Debugf("connecting to nats...")
		natsAddr := viper.GetString("nats-addr")
		natsInstance, err := persistence.NewNats(
			persistence.NatsConnectionOpts{
				AppName: appName,
				Host:    natsAddr,
			},
			persistence.NatsAuthOpts{
				NKey: viper.GetString("nats-nkey-value"),
			},
			&serviceLogs,
		)
		if err != nil {
			return fmt.Errorf("failed to create nats client: %w", err)
		}
		if err := natsInstance.Init(); err != nil {
			return fmt.Errorf("failed to connect to nats: %w", err)
		}
		logrus.Debugf("connected to nats")
		opts.AddShutdownProcess("nats", natsInstance.Shutdown)
		audit.Log(audit.LogEntry{
			EntityId:     fmt.Sprintf("%v@%s", opts.GetUserId(), opts.GetHostname()),
			EntityType:   audit.CoordinatorEntity,
			Verb:         audit.Connect,
			ResourceId:   viper.GetString("nats-addr"),
			ResourceType: audit.DbResource,
		})
		logrus.Infof("initialised queue connection")

		//
		// cache module
		//

		logrus.Infof("initialising cache connection...")
		logrus.Debugf("connecting to redis...")
		redisAddr := viper.GetString("redis-addr")
		redisInstance := persistence.NewRedis(
			persistence.RedisConnectionOpts{
				AppName: appName,
				Addr:    redisAddr,
			},
			persistence.RedisAuthOpts{
				Username: viper.GetString("redis-username"),
				Password: viper.GetString("redis-password"),
			},
			&serviceLogs,
		)
		if err := redisInstance.Init(); err != nil {
			return fmt.Errorf("failed to connect to redis: %w", err)
		}
		logrus.Debugf("connected to redis")
		opts.AddShutdownProcess("redis", redisInstance.Shutdown)
		audit.Log(audit.LogEntry{
			EntityId:     fmt.Sprintf("%v@%s", opts.GetUserId(), opts.GetHostname()),
			EntityType:   audit.CoordinatorEntity,
			Verb:         audit.Connect,
			ResourceId:   viper.GetString("redis-addr"),
			ResourceType: audit.DbResource,
		})
		logrus.Infof("initialised cache connection")

		controllerUrl := viper.GetString("controller-url")
		controllerApiKey := viper.GetString("controller-api-key")

		controllerHealthcheckEndpoint, _ := url.JoinPath(controllerUrl, "/api/v1/healthz")
		healthcheckProbes := []func() error{
			redisInstance.GetStatus().GetError,
			mongoInstance.GetStatus().GetError,
			natsInstance.GetStatus().GetError,
			func() error {
				req, err := http.NewRequest(http.MethodGet, controllerHealthcheckEndpoint, nil)
				if err != nil {
					return fmt.Errorf("failed to create http request: %w", err)
				}
				req.Header.Add("x-api-key", controllerApiKey)
				res, err := http.DefaultClient.Do(req)
				if err != nil {
					return fmt.Errorf("failed to execute controller healthcheck request: %w", err)
				}
				if res.Header.Get("x-api-key-status") == "deprecated" {
					logrus.Warnf("api key has been deprecated, update it asap")
				}
				if res.StatusCode != http.StatusOK {
					return fmt.Errorf("failed to receive 200 from controller, got %v", res.StatusCode)
				}
				return nil
			},
		}

		logrus.Infof("initialising web application...")
		handler, err := coordinator.GetHttpApplication(coordinator.HttpApplicationOpts{
			Cache:            redisInstance,
			ControllerApiKey: controllerApiKey,
			Queue:            natsInstance,

			LivenessChecks:  healthcheckProbes,
			ReadinessChecks: healthcheckProbes,

			ServiceLogs: opts.GetServiceLogs(),
		})
		if err != nil {
			return fmt.Errorf("failed to initialise web application: %w", err)
		}

		httpServer, err := common.NewHttpServer(common.NewHttpServerOpts{
			Addr:        viper.GetString("listen-addr"),
			Handler:     handler,
			ServiceLogs: opts.GetServiceLogs(),
		})
		if err != nil {
			return fmt.Errorf("failed to create http server: %w", err)
		}
		opts.AddShutdownProcess("http", httpServer.Shutdown)
		logrus.Infof("initialised web application")
		logrus.Infof("starting web application...")
		if err := httpServer.Start(); err != nil {
			return fmt.Errorf("failed to start http server: %w", err)
		}

		return nil
	},
})
