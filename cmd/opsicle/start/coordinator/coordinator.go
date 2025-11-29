package coordinator

import (
	"fmt"
	"opsicle/internal/audit"
	"opsicle/internal/cli"
	"opsicle/internal/common"
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

		logrus.Infof("initialising audit module...")
		logrus.Debugf("connecting to mongodb...")
		mongoInstance := persistence.NewMongo(
			persistence.MongoConnectionOpts{
				AppName:  appName,
				Hosts:    viper.GetStringSlice("mongo-host"),
				IsDirect: true,
			},
			persistence.MongoAuthOpts{
				Password: viper.GetString("mongo-password"),
				Username: viper.GetString("mongo-user"),
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
		logrus.Infof("connected to nats")
		opts.AddShutdownProcess("nats", natsInstance.Shutdown)
		audit.Log(audit.LogEntry{
			EntityId:     fmt.Sprintf("%v@%s", opts.GetUserId(), opts.GetHostname()),
			EntityType:   audit.CoordinatorEntity,
			Verb:         audit.Connect,
			ResourceId:   viper.GetString("nats-addr"),
			ResourceType: audit.DbResource,
		})

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
		logrus.Infof("connected to redis")
		opts.AddShutdownProcess("redis", redisInstance.Shutdown)
		audit.Log(audit.LogEntry{
			EntityId:     fmt.Sprintf("%v@%s", opts.GetUserId(), opts.GetHostname()),
			EntityType:   audit.CoordinatorEntity,
			Verb:         audit.Connect,
			ResourceId:   viper.GetString("redis-addr"),
			ResourceType: audit.DbResource,
		})

		healthcheckProbes := []func() error{
			redisInstance.GetStatus().GetError,
			mongoInstance.GetStatus().GetError,
			natsInstance.GetStatus().GetError,
		}

		handler := coordinator.GetHttpApplication(coordinator.HttpApplicationOpts{
			Cache: redisInstance,
			Queue: natsInstance,

			LivenessChecks:  healthcheckProbes,
			ReadinessChecks: healthcheckProbes,

			ServiceLogs: opts.GetServiceLogs(),
		})
		httpServer, err := common.NewHttpServer(common.NewHttpServerOpts{
			Addr:        viper.GetString("listen-addr"),
			Handler:     handler,
			ServiceLogs: opts.GetServiceLogs(),
		})
		if err != nil {
			return fmt.Errorf("failed to create http server: %w", err)
		}
		opts.AddShutdownProcess("http", httpServer.Shutdown)
		if err := httpServer.Start(); err != nil {
			return fmt.Errorf("failed to start http server: %w", err)
		}

		return nil
	},
})
