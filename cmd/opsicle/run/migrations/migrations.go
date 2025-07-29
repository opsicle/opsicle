package migrations

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/database"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
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
		DefaultValue: 3306,
		Usage:        "specifies the port which the database is listening on",
		Type:         cli.FlagTypeInteger,
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
		Name:         "steps",
		Short:        's',
		DefaultValue: 0,
		Usage:        "when this is non-zero, only that number of steps will be applied",
		Type:         cli.FlagTypeInteger,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "migrations",
	Short: "Runs any database migrations",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)

		databaseConnection, err := database.ConnectMysql(database.ConnectOpts{
			ConnectionId: "opsicle/migrator",
			Host:         viper.GetString("db-host"),
			Port:         viper.GetInt("db-port"),
			Username:     viper.GetString("db-user"),
			Password:     viper.GetString("db-password"),
			Database:     viper.GetString("db-name"),
		})
		if err != nil {
			return fmt.Errorf("failed to connect to mysql: %s", err)
		}
		steps := viper.GetInt("steps")
		if err := database.MigrateMysql(database.MigrateOpts{
			Connection:  databaseConnection,
			Steps:       steps,
			ServiceLogs: serviceLogs,
		}); err != nil {
			return fmt.Errorf("failed to migrate mysql: %s", err)
		}

		<-time.After(500 * time.Millisecond)
		logrus.Infof("database migration successful")
		return nil
	},
}
