package controller

import (
	"fmt"
	"net"
	"opsicle/internal/audit"
	"opsicle/internal/cache"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/controller"
	"opsicle/internal/email"
	"opsicle/internal/persistence"
	"opsicle/internal/queue"
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

		logrus.Infof("establishing connection to audit database...")
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
			EntityType:   audit.ControllerEntity,
			Verb:         audit.Connect,
			ResourceId:   fmt.Sprintf("%s:%v", viper.GetString("mongo-host"), viper.GetInt("mongo-port")),
			ResourceType: audit.DbResource,
		})

		/*
		  ___ _      _ _____ ___ ___  ___ __  __   ___   _ _____ _   ___   _   ___ ___
		 | _ | |    /_|_   _| __/ _ \| _ |  \/  | |   \ /_|_   _/_\ | _ ) /_\ / __| __|
		 |  _| |__ / _ \| | | _| (_) |   | |\/| | | |) / _ \| |/ _ \| _ \/ _ \\__ | _|
		 |_| |____/_/ \_|_| |_| \___/|_|_|_|  |_| |___/_/ \_|_/_/ \_|___/_/ \_|___|___|

		*/

		logrus.Infof("establishing connection to platform database...")
		logrus.Debugf("connecting to mysql...")
		host := viper.GetString("mysql-host")
		port := viper.GetInt("mysql-port")

		addr := net.JoinHostPort(host, strconv.Itoa(port))
		mysqlInstance := persistence.NewMysql(
			persistence.MysqlConnectionOpts{
				AppName:  appName,
				Host:     addr,
				Database: viper.GetString("mysql-database"),
			},
			persistence.MysqlAuthOpts{
				Password: viper.GetString("mysql-password"),
				Username: viper.GetString("mysql-user"),
			},
			nil,
		)
		if err := mysqlInstance.Init(); err != nil {
			return fmt.Errorf("failed to connect to mysql: %w", err)
		}
		logrus.Infof("connected to mysql")
		opts.AddShutdownProcess("mysql", mysqlInstance.Shutdown)
		audit.Log(audit.LogEntry{
			EntityId:     fmt.Sprintf("%v@%s", opts.GetUserId(), opts.GetHostname()),
			EntityType:   audit.ControllerEntity,
			Verb:         audit.Execute,
			ResourceId:   fmt.Sprintf("%s:%v", viper.GetString("mysql-host"), viper.GetInt("mysql-port")),
			ResourceType: audit.DbResource,
		})

		/*
		   ___   _   ___ _  _ ___
		  / __| /_\ / __| || | __|
		 | (__ / _ | (__| __ | _|
		  \___/_/ \_\___|_||_|___|

		*/

		logrus.Infof("establishing connection to cache...")
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
			EntityType:   audit.ControllerEntity,
			Verb:         audit.Connect,
			ResourceId:   viper.GetString("redis-addr"),
			ResourceType: audit.CacheResource,
		})
		if err := cache.InitRedis(cache.InitRedisOpts{
			RedisConnection: redisInstance,
			ServiceLogs:     serviceLogs,
		}); err != nil {
			return fmt.Errorf("failed to initialise redis cache: %w", err)
		}

		/*
		   ___  _   _ ___ _   _ ___
		  / _ \| | | | __| | | | __|
		 | (_) | |_| | _|| |_| | _|
		  \__\_\\___/|___|\___/|___|

		*/
		logrus.Infof("establishing connection to queue...")
		queueId := "controller"
		nats, err := queue.InitNats(queue.InitNatsOpts{
			Id:          queueId,
			Addr:        viper.GetString("nats-addr"),
			Username:    viper.GetString("nats-username"),
			Password:    viper.GetString("nats-password"),
			NKey:        viper.GetString("nats-nkey-value"),
			ServiceLogs: serviceLogs,
		})
		if err != nil {
			return fmt.Errorf("failed to initialise nats queue: %w", err)
		}
		if err := nats.Connect(); err != nil {
			return fmt.Errorf("failed to connect to nats: %w", err)
		}
		logrus.Debugf("established connection to queue")
		audit.Log(audit.LogEntry{
			EntityId:     fmt.Sprintf("%v@%s", opts.GetUserId(), opts.GetHostname()),
			EntityType:   audit.ControllerEntity,
			Verb:         audit.Connect,
			ResourceId:   viper.GetString("nats-addr"),
			ResourceType: audit.CacheResource,
		})

		logrus.Infof("initialising application...")

		healthcheckProbes := []func() error{
			mysqlInstance.GetStatus().GetError,
			mongoInstance.GetStatus().GetError,
		}

		sessionSigningToken := viper.GetString("session-signing-token")
		controllerOpts := controller.HttpApplicationOpts{
			DatabaseConnection:  mysqlInstance,
			QueueId:             queueId,
			ReadinessChecks:     healthcheckProbes,
			LivenessChecks:      healthcheckProbes,
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
		logrus.Infof("initialised email")

		logrus.Infof("initialising application server...")
		httpServerDone := make(chan common.Done)
		listenAddress := viper.GetString("listen-addr")
		publicUrl := viper.GetString("public-server-url")
		if publicUrl == "" {
			publicUrl = fmt.Sprintf("http://%s", listenAddress)
		}
		if err := controller.SetPublicServerUrl(publicUrl); err != nil {
			return fmt.Errorf("failed to set the public url: %w", err)
		}
		controllerHandler := controller.GetHttpApplication(controllerOpts)
		logrus.Debugf("initialised application")

		server, err := common.NewHttpServer(common.NewHttpServerOpts{
			Addr:        listenAddress,
			Done:        httpServerDone,
			Handler:     controllerHandler,
			ServiceLogs: serviceLogs,
		})
		if err != nil {
			return fmt.Errorf("failed to create new http server: %w", err)
		}
		logrus.Debugf("initialised server")
		logrus.Infof("starting server...")
		opts.AddShutdownProcess("http", server.Shutdown)
		if err := server.Start(); err != nil {
			return fmt.Errorf("failed to start http server: %w", err)
		}
		return nil
	},
})
