package audit_database

import (
	"context"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/database"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "mongo-host",
		Short:        'H',
		DefaultValue: "127.0.0.1",
		Usage:        "Specifies the hostname of the MongoDB instance",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-port",
		Short:        'P',
		DefaultValue: "27017",
		Usage:        "Specifies the port which the MongoDB instance is listening on",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-user",
		Short:        'U',
		DefaultValue: "opsicle",
		Usage:        "Specifies the username to use to login to the MongoDB instance",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-password",
		Short:        'p',
		DefaultValue: "password",
		Usage:        "Specifies the password to use to login to the MongoDB instance",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "audit-database",
	Aliases: []string{"adb"},
	Short:   "Checks database connectivity with the audit database (MongoDB)",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("verifying audit database connectivity...")
		connectionId := "opsicle/check/audit_database"
		databaseConnection, err := database.ConnectMongo(database.ConnectOpts{
			ConnectionId: connectionId,
			Host:         viper.GetString("mongo-host"),
			Port:         viper.GetInt("mongo-port"),
			Username:     viper.GetString("mongo-user"),
			Password:     viper.GetString("mongo-password"),
		})
		if err != nil {
			return fmt.Errorf("failed to establish connection to database: %w", err)
		}
		if err = databaseConnection.Ping(context.TODO(), nil); err != nil {
			return fmt.Errorf("failed to ping database: %w", err)
		}
		cli.PrintBoxedSuccessMessage(fmt.Sprintf(
			"Successfully connected to audit database at url[%s:%v]",
			viper.GetString("mongo-host"),
			viper.GetInt("mongo-port"),
		))
		return nil
	},
}
