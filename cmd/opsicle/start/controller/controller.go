package controller

import (
	"fmt"
	"opsicle/internal/audit"
	"opsicle/internal/cache"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/controller"
	"opsicle/internal/database"
	"opsicle/internal/email"
	"opsicle/internal/queue"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "admin-api-token",
		DefaultValue: "",
		Usage:        "specify this to enable usage of the admin endpoints, send this in the Authorization header as a Bearer token",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "listen-addr",
		DefaultValue: "0.0.0.0:54321",
		Usage:        "specifies the listen address of the server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-host",
		DefaultValue: "127.0.0.1",
		Usage:        "Specifies the hostname of the MongoDB instance",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-port",
		DefaultValue: "27017",
		Usage:        "Specifies the port which the MongoDB instance is listening on",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-user",
		DefaultValue: "opsicle",
		Usage:        "Specifies the username to use to login to the MongoDB instance",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-password",
		DefaultValue: "password",
		Usage:        "Specifies the password to use to login to the MongoDB instance",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-host",
		DefaultValue: "127.0.0.1",
		Usage:        "specifies the hostname of the database",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-port",
		DefaultValue: "3306",
		Usage:        "specifies the port which the database is listening on",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-database",
		DefaultValue: "opsicle",
		Usage:        "specifies the name of the central database schema",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-user",
		DefaultValue: "opsicle",
		Usage:        "specifies the username to use to login",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-password",
		DefaultValue: "password",
		Usage:        "specifies the password to use to login",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "nats-addr",
		DefaultValue: "localhost:4222",
		Usage:        "Specifies the hostname (including port) of the NATS server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "nats-username",
		DefaultValue: "opsicle",
		Usage:        "Specifies the username used to login to NATS",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "nats-password",
		DefaultValue: "password",
		Usage:        "Specifies the password used to login to NATS",
		Type:         cli.FlagTypeString,
	},
	{
		Name: "nats-nkey-value",
		// this default value is the development nkey, this value must be aligned
		// to the one in `./docker-compose.yml` in the root of the repository
		DefaultValue: "SUADZTA4VJHBCO7K75DQ3IN7KZGWHKEI26D2IYEABRN5TXXYHXLWNDYT4A",
		Usage:        "Specifies the nkey used to login to NATS",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "public-server-url",
		DefaultValue: "",
		Usage:        "specifies a url where the controller server can be accessed via - required for emails to work properly",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "redis-addr",
		DefaultValue: "localhost:6379",
		Usage:        "defines the hostname (including port) of the redis server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "redis-username",
		DefaultValue: "opsicle",
		Usage:        "defines the username used to login to redis",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "redis-password",
		DefaultValue: "password",
		Usage:        "defines the password used to login to redis",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "sender-email",
		DefaultValue: "noreply@notification.opsicle.io",
		Usage:        "defines the notification sender's address",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "sender-name",
		DefaultValue: "Opsicle Notifications",
		Usage:        "defines the notification sender's name",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "session-signing-token",
		DefaultValue: "super_secret_session_signing_token",
		Usage:        "specifies the token used to sign sessions",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "smtp-username",
		DefaultValue: "noreply@notification.opsicle.io",
		Usage:        "defines the smtp server user's email address",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "smtp-password",
		DefaultValue: "",
		Usage:        "defines the smtp server user's password",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "smtp-hostname",
		DefaultValue: "smtp.eu.mailgun.org",
		Usage:        "defines the smtp server's hostname",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "smtp-port",
		DefaultValue: 587,
		Usage:        "defines the smtp server's port",
		Type:         cli.FlagTypeInteger,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "controller",
	Aliases: []string{"c"},
	Short:   "Starts the controller component",
	Long:    "Starts the controller component which serves as the API layer that user interfaces can connect to to perform actions",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Debugf("starting logging engine...")
		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)
		logrus.Debugf("started logging engine")

		connectionId := "opsicle/controller"

		/*
		    _  _   _ ___ ___ _____   ___   _ _____ _   ___   _   ___ ___
		   /_\| | | |   |_ _|_   _| |   \ /_|_   _/_\ | _ ) /_\ / __| __|
		  / _ | |_| | |) | |  | |   | |) / _ \| |/ _ \| _ \/ _ \\__ | _|
		 /_/ \_\___/|___|___| |_|   |___/_/ \_|_/_/ \_|___/_/ \_|___|___|

		*/

		logrus.Infof("establishing connection to audit database...")
		auditDatabaseConnection, err := database.ConnectMongo(database.ConnectOpts{
			ConnectionId: connectionId,
			Host:         viper.GetString("mongo-host"),
			Port:         viper.GetInt("mongo-port"),
			Username:     viper.GetString("mongo-user"),
			Password:     viper.GetString("mongo-password"),
		})
		if err != nil {
			return fmt.Errorf("failed to establish connection to audit database: %w", err)
		}
		logrus.Debugf("established connection to audit database")
		logrus.Infof("starting audit database connection freshness verifier...")
		auditDatabaseConnectionOk := false
		auditDatabaseConnectionStatusLastUpdatedAt := time.Now()
		auditDatabaseConnectionStatusUpdates := make(chan bool)
		var auditDatabaseConnectionStatusMutex sync.Mutex
		var auditModuleError error = nil
		var auditModuleErrorMutex sync.Mutex
		go func() {
			for {
				if auditModuleError == nil {
					logrus.Trace("audit module is ok")
					<-time.After(3 * time.Second)
					continue
				}
				if auditDatabaseConnectionOk {
					logrus.Tracef("(re)trying initialisation of audit module (last error: %s)...", auditModuleError)
					auditModuleErrorMutex.Lock()
					auditModuleError = audit.InitMongo(auditDatabaseConnection)
					if auditModuleError != nil {
						logrus.Errorf("failed to initialise audit module: %s", auditModuleError)
					}
					auditModuleErrorMutex.Unlock()
				} else {
					logrus.Tracef("audit module is not ok (error: %s), waiting for audit database restoration...", auditModuleError)
				}
				<-time.After(3 * time.Second)
			}
		}()
		go func() {
			for {
				statusUpdate := <-auditDatabaseConnectionStatusUpdates
				auditDatabaseConnectionStatusMutex.Lock()
				if statusUpdate != auditDatabaseConnectionOk {
					logAtLevel := logrus.Infof
					if !statusUpdate {
						logAtLevel = logrus.Warnf
						auditModuleError = fmt.Errorf("database connection lost")
					}
					logAtLevel("audit database connection freshness status switched to '%v'", statusUpdate)
					auditDatabaseConnectionStatusLastUpdatedAt = time.Now()
				}
				auditDatabaseConnectionOk = statusUpdate
				auditDatabaseConnectionStatusMutex.Unlock()
			}
		}()
		go func() {
			for {
				logrus.Tracef("verifying audit database connection freshness...")
				if err := database.CheckMongoConnection(connectionId); err != nil {
					logrus.Errorf("failed to check mongo connection with id '%s': %s", connectionId, err)
					auditDatabaseConnectionStatusUpdates <- false
					if err := database.RefreshMongoConnection(connectionId); err != nil {
						logrus.Errorf("failed to refresh mongo connection with id '%s': %s", connectionId, err)
					} else {
						if err := audit.InitMongo(auditDatabaseConnection); err != nil {
							logrus.Errorf("failed to re-initialise audit module: %s", err)
						}
					}
				} else {
					logrus.Tracef("audit database connection freshness verified")
					auditDatabaseConnectionStatusUpdates <- true
				}
				<-time.After(3 * time.Second)
			}
		}()
		if auditModuleError = audit.InitMongo(auditDatabaseConnection); auditModuleError != nil {
			return fmt.Errorf("failed to initialise audit module: %w", auditModuleError)
		}
		hostname, _ := os.Hostname()
		userId := os.Getuid()
		audit.Log(audit.LogEntry{
			EntityId:     fmt.Sprintf("%v@%s", userId, hostname),
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
		databaseConnection, err := database.ConnectMysql(database.ConnectOpts{
			ConnectionId: connectionId,
			Host:         viper.GetString("mysql-host"),
			Port:         viper.GetInt("mysql-port"),
			Username:     viper.GetString("mysql-user"),
			Password:     viper.GetString("mysql-password"),
			Database:     viper.GetString("mysql-database"),
		})
		if err != nil {
			return fmt.Errorf("failed to establish connection to platform database: %w", err)
		}
		logrus.Debugf("established connection to platform database")
		logrus.Infof("starting platform database connection freshness verifier...")
		platformDatabaseConnectionOk := false
		platformDatabaseConnectionStatusLastUpdatedAt := time.Now()
		platformDatabaseConnectionStatusUpdates := make(chan bool)
		var platformDatabaseConnectionStatusMutex sync.Mutex
		go func() {
			for {
				statusUpdate := <-platformDatabaseConnectionStatusUpdates
				platformDatabaseConnectionStatusMutex.Lock()
				if statusUpdate != platformDatabaseConnectionOk {
					logAtLevel := logrus.Infof
					if !statusUpdate {
						logAtLevel = logrus.Warnf
					}
					logAtLevel("platform database connection freshness status switched to '%v'", statusUpdate)
					platformDatabaseConnectionStatusLastUpdatedAt = time.Now()
				}
				platformDatabaseConnectionOk = statusUpdate
				platformDatabaseConnectionStatusMutex.Unlock()
			}
		}()
		go func() {
			for {
				logrus.Tracef("verifying platform database connection freshness...")
				if err := database.CheckMysqlConnection(connectionId); err != nil {
					logrus.Errorf("failed to check mysql connection with id '%s': %s", connectionId, err)
					platformDatabaseConnectionStatusUpdates <- false
					if err := database.RefreshMysqlConnection(connectionId); err != nil {
						logrus.Errorf("failed to refresh mysql connection with id '%s': %s", connectionId, err)
					}
				} else {
					logrus.Tracef("platform database connection freshness verified")
					platformDatabaseConnectionStatusUpdates <- true
				}
				<-time.After(3 * time.Second)
			}
		}()
		audit.Log(audit.LogEntry{
			EntityId:     fmt.Sprintf("%v@%s", userId, hostname),
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
		if err := cache.InitRedis(cache.InitRedisOpts{
			Addr:        viper.GetString("redis-addr"),
			Username:    viper.GetString("redis-username"),
			Password:    viper.GetString("redis-password"),
			ServiceLogs: serviceLogs,
		}); err != nil {
			return fmt.Errorf("failed to initialise redis cache: %w", err)
		}
		logrus.Debugf("established connection to cache")
		audit.Log(audit.LogEntry{
			EntityId:     fmt.Sprintf("%v@%s", userId, hostname),
			EntityType:   audit.ControllerEntity,
			Verb:         audit.Connect,
			ResourceId:   viper.GetString("redis-addr"),
			ResourceType: audit.CacheResource,
		})

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
			EntityId:     fmt.Sprintf("%v@%s", userId, hostname),
			EntityType:   audit.ControllerEntity,
			Verb:         audit.Connect,
			ResourceId:   viper.GetString("nats-addr"),
			ResourceType: audit.CacheResource,
		})

		logrus.Infof("initialising application...")

		sessionSigningToken := viper.GetString("session-signing-token")
		controllerOpts := controller.HttpApplicationOpts{
			DatabaseConnection: databaseConnection,
			QueueId:            queueId,
			ReadinessChecks: []func() error{
				func() error {
					if !auditDatabaseConnectionOk {
						return fmt.Errorf("audit database connection is pending restoration")
					}
					return nil
				},
				func() error {
					if !platformDatabaseConnectionOk {
						return fmt.Errorf("platform database connection is pending restoration")
					}
					return nil
				},
			},
			LivenessChecks: []func() error{
				func() error {
					if !auditDatabaseConnectionOk && auditDatabaseConnectionStatusLastUpdatedAt.Before(time.Now().Add(-30*time.Second)) {
						return fmt.Errorf("audit database connection is invalid")
					}
					return nil
				},
				func() error {
					if !platformDatabaseConnectionOk && platformDatabaseConnectionStatusLastUpdatedAt.Before(time.Now().Add(-30*time.Second)) {
						return fmt.Errorf("platform database connection is invalid")
					}
					return nil
				},
			},
			ServiceLogs:         serviceLogs,
			SessionSigningToken: sessionSigningToken,
		}
		adminToken := viper.GetString("admin-api-token")
		if adminToken != "" {
			logrus.Infof("initialising admin endpoints...")
			if len(adminToken) < 36 {
				return fmt.Errorf("admin token must be 36 characters or longer for security purposes (hint: use a uuid)")
			}
			controllerOpts.AdminToken = adminToken
			logrus.Infof("admin endpoints are available")
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
		if err := server.Start(); err != nil {
			return fmt.Errorf("failed to start http server: %w", err)
		}
		return nil
	},
}
