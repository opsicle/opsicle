package database

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/database"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "mysql-host",
		Short:        'H',
		DefaultValue: "127.0.0.1",
		Usage:        "Specifies the hostname of the MySQL instance",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-port",
		Short:        'P',
		DefaultValue: "3306",
		Usage:        "Specifies the port which the MySQL instance is listening on",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-database",
		Short:        'N',
		DefaultValue: "opsicle",
		Usage:        "Specifies the name of the MySQL central database schema",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-user",
		Short:        'U',
		DefaultValue: "opsicle",
		Usage:        "Specifies the username to use to login to the MySQL instance",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-password",
		Short:        'p',
		DefaultValue: "password",
		Usage:        "Specifies the password to use to login to the MySQL instance",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "database",
	Aliases: []string{"db"},
	Short:   "Checks database connectivity with the platform database (MySQL)",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("verifying platform database connectivity...")
		connectionId := "opsicle/check/database"
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
		if err = databaseConnection.Ping(); err != nil {
			return fmt.Errorf("failed to ping database: %w", err)
		}
		_, err = databaseConnection.Exec("SELECT 1")
		if err != nil {
			return fmt.Errorf("failed to send test query to database: %w", err)
		}
		defer databaseConnection.Close()
		cli.PrintBoxedSuccessMessage(fmt.Sprintf(
			"Successfully connected to database at url[%s:%v]",
			viper.GetString("mysql-host"),
			viper.GetInt("mysql-port"),
		))
		return nil
	},
}
