package controller

import (
	"fmt"
	"opsicle/internal/cache"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/controller"
	"opsicle/internal/database"
	"opsicle/internal/email"
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
		Name:         "mysql-host",
		Short:        'H',
		DefaultValue: "127.0.0.1",
		Usage:        "specifies the hostname of the database",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-port",
		Short:        'P',
		DefaultValue: "3306",
		Usage:        "specifies the port which the database is listening on",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-database",
		Short:        'N',
		DefaultValue: "opsicle",
		Usage:        "specifies the name of the central database schema",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-user",
		Short:        'U',
		DefaultValue: "opsicle",
		Usage:        "specifies the username to use to login",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-password",
		Short:        'p',
		DefaultValue: "password",
		Usage:        "specifies the password to use to login",
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

		logrus.Infof("establishing connection to database...")
		connectionId := "opsicle/controller"
		databaseConnection, err := database.ConnectMysql(database.ConnectOpts{
			ConnectionId: connectionId,
			Host:         viper.GetString("mysql-host"),
			Port:         viper.GetInt("mysql-port"),
			Username:     viper.GetString("mysql-user"),
			Password:     viper.GetString("mysql-password"),
			Database:     viper.GetString("mysql-database"),
		})
		if err != nil {
			return fmt.Errorf("failed to establish connection to database: %w", err)
		}
		logrus.Debugf("established connection to database")
		logrus.Infof("starting connection freshness verifier...")
		databaseConnectionOk := true
		databaseConnectionStatusLastUpdatedAt := time.Now()
		databaseConnectionStatusUpdates := make(chan bool)
		var databaseConnectionStatusMutex sync.Mutex
		go func() {
			for {
				statusUpdate := <-databaseConnectionStatusUpdates
				databaseConnectionStatusMutex.Lock()
				if statusUpdate != databaseConnectionOk {
					logAtLevel := logrus.Infof
					if !statusUpdate {
						logAtLevel = logrus.Warnf
					}
					logAtLevel("database connection freshness status switched to '%v'", statusUpdate)
					databaseConnectionStatusLastUpdatedAt = time.Now()
				}
				databaseConnectionOk = statusUpdate
				databaseConnectionStatusMutex.Unlock()
			}
		}()
		go func() {
			for {
				logrus.Tracef("verifying database connection freshness...")
				if err := database.CheckMysqlConnection(connectionId); err != nil {
					logrus.Errorf("failed to check mysql connection with id '%s': %s", connectionId, err)
					databaseConnectionStatusUpdates <- false
					if err := database.RefreshMysqlConnection(connectionId); err != nil {
						logrus.Errorf("failed to refresh mysql connection with id '%s': %s", connectionId, err)
					}
				} else {
					logrus.Tracef("database connection freshness verified")
					databaseConnectionStatusUpdates <- true
				}
				<-time.After(3 * time.Second)
			}
		}()

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

		logrus.Infof("initialising application...")

		sessionSigningToken := viper.GetString("session-signing-token")
		controllerOpts := controller.HttpApplicationOpts{
			DatabaseConnection: databaseConnection,
			ReadinessChecks: []func() error{
				func() error {
					if !databaseConnectionOk {
						return fmt.Errorf("database connection is pending restoration")
					}
					return nil
				},
			},
			LivenessChecks: []func() error{
				func() error {
					if !databaseConnectionOk && databaseConnectionStatusLastUpdatedAt.Before(time.Now().Add(-30*time.Second)) {
						return fmt.Errorf("database connection is invalid")
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
