package queue

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/queue"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "nats-addr",
		DefaultValue: "localhost:4222",
		Usage:        "Specifies the hostname (including port) of the NATS server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "nats-username",
		DefaultValue: "opsicle",
		Usage:        "Specifies the username used to login to NATS",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "nats-password",
		DefaultValue: "password",
		Usage:        "Specifies the password used to login to NATS",
		Type:         cli.FlagTypeString,
	},
	{
		Name: "nats-nkey",
		// this default value is the development nkey, this value must be aligned
		// to the one in `./docker-compose.yml` in the root of the repository
		DefaultValue: "SUADZTA4VJHBCO7K75DQ3IN7KZGWHKEI26D2IYEABRN5TXXYHXLWNDYT4A",
		Usage:        "Specifies the nkey used to login to NATS",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "queue",
	Aliases: []string{"q"},
	Short:   "Checks queue connectivity",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("verifying queue connectivity...")
		nkey := viper.GetString("nats-nkey")
		username := viper.GetString("nats-username")
		password := viper.GetString("nats-password")
		queueOpts := queue.InitNatsOpts{
			Addr: viper.GetString("nats-addr"),
		}
		if nkey != "" {
			queueOpts.NKey = nkey
		} else if username != "" && password != "" {
			queueOpts.Username = username
			queueOpts.Password = password
		}

		connection, err := queue.InitNats(queueOpts)
		if err != nil {
			return fmt.Errorf("failed to initialise nats queue: %w", err)
		}
		defer connection.Drain()
		defer connection.Close()
		cli.PrintBoxedSuccessMessage(fmt.Sprintf(
			"Successfully connected to queue at address[%s]",
			viper.GetString("nats-addr"),
		))
		return nil
	},
}
