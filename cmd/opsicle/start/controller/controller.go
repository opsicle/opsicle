package controller

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/controller"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
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
		serviceLogs := make(chan common.ServiceLog, 64)
		httpServerDone := make(chan common.Done)
		common.StartServiceLogLoop(serviceLogs)

		controllerHandler := controller.GetHttpApplication(serviceLogs)

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
		if err := server.Start(); err != nil {
			return fmt.Errorf("failed to start http server: %s", err)
		}
		return cmd.Help()
	},
}
