package controller

import (
	"fmt"
	"net"
	"opsicle/internal/audit"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/config"
	"opsicle/internal/controller"
	"opsicle/internal/email"
	"opsicle/internal/persistence"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Command = cli.NewCommand(cli.CommandOpts{
	Flags:   flags,
	Use:     "controller",
	Aliases: []string{"c"},
	Short:   "Starts the controller component",
	Long:    "Starts the controller component which serves as the API layer that user interfaces can connect to to perform actions",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		appName := opts.GetFullname()
		serviceLogs := opts.GetServiceLogs()

		/*
		    _  _   _ ___ ___ _____   ___   _ _____ _   ___   _   ___ ___
		   /_\| | | |   |_ _|_   _| |   \ /_|_   _/_\ | _ ) /_\ / __| __|
		  / _ | |_| | |) | |  | |   | |) / _ \| |/ _ \| _ \/ _ \\__ | _|
		 /_/ \_\___/|___|___| |_|   |___/_/ \_|_/_/ \_|___/_/ \_|___|___|

		*/

		logrus.Infof("audit database initialising...")
		logrus.Debugf("connecting to mongodb...")
		mongoInstance := persistence.NewMongo(
			persistence.MongoConnectionOpts{
				AppName:  appName,
				Hosts:    viper.GetStringSlice(config.MongoHost),
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
		logrus.Debugf("connected to mongodb")
		if auditModuleError := audit.InitMongo(mongoInstance.GetClient()); auditModuleError != nil {
			return fmt.Errorf("failed to initialise audit module: %w", auditModuleError)
		}
		opts.AddShutdownProcess("mongo", mongoInstance.Shutdown)
		audit.Log(audit.LogEntry{
			EntityId:     fmt.Sprintf("%v@%s", opts.GetUserId(), opts.GetHostname()),
			EntityType:   audit.ControllerEntity,
			Verb:         audit.Connect,
			ResourceId:   fmt.Sprintf("%s:%v", viper.GetString("mongo-host"), viper.GetInt("mongo-port")),
			ResourceType: audit.DbResource,
		})
		logrus.Infof("audit database initialised")

		/*
		  ___ _      _ _____ ___ ___  ___ __  __   ___   _ _____ _   ___   _   ___ ___
		 | _ | |    /_|_   _| __/ _ \| _ |  \/  | |   \ /_|_   _/_\ | _ ) /_\ / __| __|
		 |  _| |__ / _ \| | | _| (_) |   | |\/| | | |) / _ \| |/ _ \| _ \/ _ \\__ | _|
		 |_| |____/_/ \_|_| |_| \___/|_|_|_|  |_| |___/_/ \_|_/_/ \_|___/_/ \_|___|___|

		*/

		logrus.Infof("platform database initialising...")
		logrus.Debugf("connecting to mysql...")
		mysqlHost := viper.GetString(config.MysqlHost)
		mysqlPort := viper.GetInt(config.MysqlPort)
		addr := net.JoinHostPort(mysqlHost, strconv.Itoa(mysqlPort))
		mysqlInstance := persistence.NewMysql(
			persistence.MysqlConnectionOpts{
				AppName:  appName,
				Host:     addr,
				Database: viper.GetString(config.MysqlDatabase),
			},
			persistence.MysqlAuthOpts{
				Password: viper.GetString(config.MysqlPassword),
				Username: viper.GetString(config.MysqlUsername),
			},
			nil,
		)
		if err := mysqlInstance.Init(); err != nil {
			return fmt.Errorf("failed to connect to mysql: %w", err)
		}
		logrus.Debugf("connected to mysql")
		opts.AddShutdownProcess("mysql", mysqlInstance.Shutdown)
		audit.Log(audit.LogEntry{
			EntityId:     fmt.Sprintf("%v@%s", opts.GetUserId(), opts.GetHostname()),
			EntityType:   audit.ControllerEntity,
			Verb:         audit.Execute,
			ResourceId:   fmt.Sprintf("%s:%v", mysqlHost, mysqlPort),
			ResourceType: audit.DbResource,
		})
		logrus.Infof("platform database initialised")

		/*
		   ___   _   ___ _  _ ___
		  / __| /_\ / __| || | __|
		 | (__ / _ | (__| __ | _|
		  \___/_/ \_\___|_||_|___|

		*/

		logrus.Infof("cache initialising...")
		logrus.Debugf("connecting to redis...")
		redisAddr := viper.GetString(config.RedisAddr)
		redisInstance := persistence.NewRedis(
			persistence.RedisConnectionOpts{
				AppName: appName,
				Addr:    redisAddr,
			},
			persistence.RedisAuthOpts{
				Username: viper.GetString(config.RedisUsername),
				Password: viper.GetString(config.RedisPassword),
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
			EntityType:   audit.ControllerEntity,
			Verb:         audit.Connect,
			ResourceId:   viper.GetString(config.RedisAddr),
			ResourceType: audit.CacheResource,
		})
		logrus.Infof("cache initialised")

		/*
		   ___  _   _ ___ _   _ ___
		  / _ \| | | | __| | | | __|
		 | (_) | |_| | _|| |_| | _|
		  \__\_\\___/|___|\___/|___|

		*/
		logrus.Infof("queue initialising...")
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
		logrus.Infof("queue initialised")

		healthcheckProbes := []func() error{
			mysqlInstance.GetStatus().GetError,
			mongoInstance.GetStatus().GetError,
		}

		sessionSigningToken := viper.GetString("session-signing-token")
		apiKeys := viper.GetStringSlice("api-keys")
		listenAddress := viper.GetString("listen-addr")
		publicUrl := viper.GetString("public-server-url")
		if publicUrl == "" {
			publicUrl = fmt.Sprintf("http://%s", listenAddress)
		}

		controllerOpts := controller.HttpApplicationOpts{
			ApiKeys:             apiKeys,
			CacheConnection:     redisInstance,
			DatabaseConnection:  mysqlInstance,
			ReadinessChecks:     healthcheckProbes,
			LivenessChecks:      healthcheckProbes,
			PublicServerUrl:     publicUrl,
			QueueConnection:     natsInstance,
			ServiceLogs:         serviceLogs,
			SessionSigningToken: sessionSigningToken,
		}

		logrus.Infof("initialising email...")
		smtpHost := viper.GetString("smtp-hostname")
		smtpPort := viper.GetInt("smtp-port")
		smtpUsername := viper.GetString("smtp-username")
		smtpPassword := viper.GetString("smtp-password")
		senderEmail := viper.GetString("sender-email")
		senderName := viper.GetString("sender-name")
		controllerOpts.EmailConfig = &controller.SmtpServerConfig{
			Hostname: smtpHost,
			Port:     smtpPort,
			Username: smtpUsername,
			Password: smtpPassword,
			Sender: email.User{
				Address: senderEmail,
				Name:    senderName,
			},
		}
		logrus.Infof("email initialised")

		logrus.Infof("web application initialising...")
		controllerHandler, err := controller.GetHttpApplication(controllerOpts)
		if err != nil {
			return fmt.Errorf("failed to initialise web application: %w", err)
		}
		logrus.Infof("web application initialised")

		logrus.Infof("http server initialising...")
		server, err := common.NewHttpServer(common.NewHttpServerOpts{
			Addr:        listenAddress,
			Handler:     controllerHandler,
			ServiceLogs: serviceLogs,
		})
		if err != nil {
			return fmt.Errorf("failed to create new http server: %w", err)
		}
		logrus.Infof("http server initialised")

		logrus.Infof("starting controller component...")
		opts.AddShutdownProcess("http", server.Shutdown)
		if err := server.Start(); err != nil {
			return fmt.Errorf("failed to start http server: %w", err)
		}
		return nil
	},
})
