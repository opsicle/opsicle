package database

import (
	"fmt"
	"net"
	"opsicle/internal/cli"
	"opsicle/internal/config"
	"opsicle/internal/persistence"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{}.
	Append(config.GetMysqlFlags())

var Command = cli.NewCommand(cli.CommandOpts{
	Name:    "utils.check.database",
	Flags:   flags,
	Use:     "database",
	Aliases: []string{"db"},
	Short:   "Checks database connectivity with the platform database (MySQL)",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		appName := opts.GetFullname()
		serviceLogs := opts.GetServiceLogs()

		logrus.Debugf("connecting to mysql...")
		mysqlHost := viper.GetString(config.MysqlHost)
		mysqlPort := viper.GetInt(config.MysqlPort)
		addr := net.JoinHostPort(mysqlHost, strconv.Itoa(mysqlPort))
		mysqlInstance := persistence.NewMysql(
			persistence.MysqlConnectionOpts{
				AppName:  appName,
				Host:     addr,
				Database: viper.GetString(config.MysqlDatabase),
			},
			persistence.MysqlAuthOpts{
				Password: viper.GetString(config.MysqlPassword),
				Username: viper.GetString(config.MysqlUsername),
			},
			&serviceLogs,
		)
		if err := mysqlInstance.Init(); err != nil {
			return fmt.Errorf("failed to connect to mysql: %w", err)
		}
		logrus.Debugf("connected to mysql")
		opts.AddShutdownProcess("mysql", mysqlInstance.Shutdown)
		cli.PrintBoxedSuccessMessage(fmt.Sprintf(
			"Successfully connected to platform database at url[%s:%v]",
			viper.GetString("mysql-host"),
			viper.GetInt("mysql-port"),
		))
		return nil
	},
})
