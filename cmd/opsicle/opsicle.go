package opsicle

import (
	"fmt"
	"opsicle/cmd/opsicle/create"
	"opsicle/cmd/opsicle/get"
	"opsicle/cmd/opsicle/list"
	"opsicle/cmd/opsicle/login"
	"opsicle/cmd/opsicle/logout"
	"opsicle/cmd/opsicle/run"
	"opsicle/cmd/opsicle/start"
	"opsicle/cmd/opsicle/validate"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/config"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var availableLogLevels = []string{
	string(common.LogLevelTrace),
	string(common.LogLevelDebug),
	string(common.LogLevelInfo),
	string(common.LogLevelWarn),
	string(common.LogLevelError),
}

var flags cli.Flags = cli.Flags{
	{
		Name:         "log-level",
		Short:        'l',
		DefaultValue: "info",
		Usage:        fmt.Sprintf("sets the log level (one of [%s])", strings.Join(availableLogLevels, ", ")),
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "config-path",
		Short:        'c',
		DefaultValue: "~/.opsicle/config",
		Usage:        "defines the location of the global configuration used",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	Command.AddCommand(create.Command)
	Command.AddCommand(get.Command)
	Command.AddCommand(list.Command)
	Command.AddCommand(login.Command)
	Command.AddCommand(logout.Command)
	Command.AddCommand(run.Command)
	Command.AddCommand(start.Command)
	Command.AddCommand(validate.Command)

	flags.AddToCommand(Command, true)

	logrus.SetOutput(os.Stderr)
	cobra.OnInitialize(func() {
		flags.BindViper(Command, true)
		cli.InitLogging(viper.GetString("log-level"))
		configPath := viper.GetString("config-path")
		logrus.Debugf("using configuration at path[%s]", configPath)
		config.LoadGlobal(configPath)
	})

	cli.InitConfig()
}

var Command = &cobra.Command{
	Use:   "opsicle",
	Short: "Runbook automations by Platform Engineers for Platform Engineers",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
