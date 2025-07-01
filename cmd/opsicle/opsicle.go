package opsicle

import (
	"fmt"
	"opsicle/cmd/opsicle/add"
	"opsicle/cmd/opsicle/run"
	"opsicle/cmd/opsicle/start"
	"opsicle/cmd/opsicle/validate"
	"opsicle/internal/config"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	Command.AddCommand(add.Command)
	Command.AddCommand(run.Command)
	Command.AddCommand(start.Command)
	Command.AddCommand(validate.Command)

	currentFlag := "log-level"
	Command.PersistentFlags().StringP(
		currentFlag,
		"l",
		"info",
		fmt.Sprintf("sets the log level (one of [%s])", strings.Join(config.LogLevels, ", ")),
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))

	currentFlag = "config-path"
	Command.PersistentFlags().StringP(
		currentFlag,
		"c",
		"~/.opsicle/config",
		"defines the location of the global configuration used",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))

	logrus.SetOutput(os.Stderr)
	cobra.OnInitialize(func() {
		logLevel := viper.GetString("log-level")
		switch logLevel {
		case config.LogLevelTrace:
			logrus.SetLevel(logrus.TraceLevel)
		case config.LogLevelDebug:
			logrus.SetLevel(logrus.DebugLevel)
		case config.LogLevelInfo:
			logrus.SetLevel(logrus.InfoLevel)
		case config.LogLevelWarn:
			logrus.SetLevel(logrus.WarnLevel)
		case config.LogLevelError:
			logrus.SetLevel(logrus.ErrorLevel)
		}
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})

		configPath := viper.GetString("config")

		config.LoadGlobal(configPath)
	})

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}

var Command = &cobra.Command{
	Use:   "opsicle",
	Short: "Runbook automations by Platform Engineers for Platform Engineers",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
