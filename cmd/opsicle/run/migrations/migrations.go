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
	{
		Name:         "force",
		DefaultValue: 0,
		Usage:        "when this is truthy, the migration will be forced to this version",
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
			return fmt.Errorf("failed to connect to mysql: %w", err)
		}
		var steps *int

		if cmd.Flags().Changed("steps") {
			inputSteps := viper.GetInt("steps")
			steps = &inputSteps
		} else {
			logrus.Infof("steps not specified, migration will be to the latest")
		}
		force := viper.GetInt("force")
		if force != 0 {
			logrus.Warnf("forcing a migration")
		}
		migrateOutput, err := database.MigrateMysql(database.MigrateOpts{
			Connection:  databaseConnection,
			Force:       force,
			Steps:       steps,
			ServiceLogs: serviceLogs,
		})
		if err != nil {
			if migrateOutput != nil {
				logrus.Infof("pre-migration database version: %v", migrateOutput.PreMigrationVersion)
				logrus.Infof("migration database dirty status: %v", migrateOutput.IsDatabaseDirty)
			}
			return fmt.Errorf("failed to migrate mysql: %w", err)
		}
		direction := "up"
		if steps != nil && *steps < 0 {
			direction = "down"
		}
		logrus.Infof("applied %v %s migrations", len(migrateOutput.VersionsApplied), direction)
		for _, version := range migrateOutput.VersionsApplied {
			logrus.Infof("  - %v", version)
		}
		logrus.Infof("migration database dirty status: %v", migrateOutput.IsDatabaseDirty)
		logrus.Infof("pre-migration database version: %v", migrateOutput.PreMigrationVersion)
		logrus.Infof("post-migration database version: %v", migrateOutput.PostMigrationVersion)

		<-time.After(500 * time.Millisecond)
		logrus.Infof("database migration operation successful")
		return nil
	},
}
