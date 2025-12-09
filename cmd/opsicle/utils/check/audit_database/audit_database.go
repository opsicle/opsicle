package audit_database

import (
	"fmt"
	"opsicle/internal/audit"
	"opsicle/internal/cli"
	"opsicle/internal/config"
	"opsicle/internal/persistence"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{}.
	Append(config.GetMongoFlags())

var Command = cli.NewCommand(cli.CommandOpts{
	Name:    "utils.check.audit_database",
	Flags:   flags,
	Use:     "audit-database",
	Aliases: []string{"adb"},
	Short:   "Checks database connectivity with the audit database (MongoDB)",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		appName := opts.GetFullname()
		serviceLogs := opts.GetServiceLogs()

		logrus.Infof("verifying audit database connectivity...")
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
		cli.PrintBoxedSuccessMessage(fmt.Sprintf(
			"Successfully connected to audit database at url[%s:%v]",
			viper.GetString("mongo-host"),
			viper.GetInt("mongo-port"),
		))
		return nil
	},
})
