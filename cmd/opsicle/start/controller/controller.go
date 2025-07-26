package controller

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/controller"
	"opsicle/internal/database"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "admin-token",
		DefaultValue: "",
		Usage:        "specify this to enable usage of the admin endpoints, send this in the Authorization header as a Bearer token",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "storage-path",
		DefaultValue: "./.opsicle",
		Usage:        "specifies the path to a directory where Opsicle data resides",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "db-host",
		Short:        'H',
		DefaultValue: "127.0.0.1",
		Usage:        "specifies the hostname of the database",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "db-port",
		Short:        'P',
		DefaultValue: "3306",
		Usage:        "specifies the port which the database is listening on",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "db-name",
		Short:        'N',
		DefaultValue: "opsicle",
		Usage:        "specifies the name of the central database schema",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "db-user",
		Short:        'U',
		DefaultValue: "opsicle",
		Usage:        "specifies the username to use to login",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "db-password",
		Short:        'p',
		DefaultValue: "password",
		Usage:        "specifies the password to use to login",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "listen-addr",
		DefaultValue: "0.0.0.0:54321",
		Usage:        "specifies the listen address of the server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "storage-mode",
		Short:        's',
		DefaultValue: common.StorageFilesystem,
		Usage:        fmt.Sprintf("specifies what type of storage we are using, one of ['%s']", strings.Join(common.Storages, "'")),
		Type:         cli.FlagTypeString,
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
		databaseConnection, err := database.ConnectMysql(database.ConnectOpts{
			ConnectionId: "opsicle/controller",
			Host:         viper.GetString("db-host"),
			Port:         viper.GetInt("db-port"),
			Username:     viper.GetString("db-user"),
			Password:     viper.GetString("db-password"),
			Database:     viper.GetString("db-name"),
		})
		if err != nil {
			return fmt.Errorf("failed to establish connection to database: %s", err)
		}
		logrus.Debugf("established connection to database")

		logrus.Infof("initialising application...")
		controllerOpts := controller.HttpApplicationOpts{
			DatabaseConnection: databaseConnection,
			ServiceLogs:        serviceLogs,
		}
		adminToken := viper.GetString("admin-token")
		if adminToken != "" {
			if len(adminToken) < 36 {
				return fmt.Errorf("admin token must be 36 characters or longer for security purposes (hint: use a uuid)")
			}
			controllerOpts.AdminToken = adminToken
		}
		controllerHandler := controller.GetHttpApplication(controllerOpts)
		logrus.Debugf("initialised application")

		logrus.Infof("initialising server...")
		httpServerDone := make(chan common.Done)
		listenAddress := viper.GetString("listen-addr")
		server, err := common.NewHttpServer(common.NewHttpServerOpts{
			Addr:        listenAddress,
			Done:        httpServerDone,
			Handler:     controllerHandler,
			ServiceLogs: serviceLogs,
		})
		if err != nil {
			return fmt.Errorf("failed to create new http server: %s", err)
		}
		logrus.Debugf("initialised server")
		logrus.Infof("starting server...")
		if err := server.Start(); err != nil {
			return fmt.Errorf("failed to start http server: %s", err)
		}
		return nil
	},
}
