package pop

import (
	"encoding/json"
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
		Name: "nats-nkey-value",
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
	Use:     "pop",
	Aliases: []string{"po"},
	Short:   "Pops a message from the configured queue",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("establishing connection to queue...")
		nats, err := queue.InitNats(queue.InitNatsOpts{
			Addr:     viper.GetString("nats-addr"),
			Username: viper.GetString("nats-username"),
			Password: viper.GetString("nats-password"),
			NKey:     viper.GetString("nats-nkey-value"),
		})
		if err != nil {
			return fmt.Errorf("failed to initialise nats queue: %w", err)
		}
		if err := nats.Connect(); err != nil {
			return fmt.Errorf("failed to connect to nats: %w", err)
		}
		logrus.Infof("established connection to queue")

		logrus.Infof("popping message from queue...")
		popOutput, err := nats.Pop(queue.PopOpts{
			Queue: queue.QueueOpts{
				Subject: "test",
				Stream:  "utils_queue",
			},
		})
		if err != nil {
			return fmt.Errorf("failed to pop message: %w", err)
		}
		if popOutput == nil {
			cli.PrintBoxedWarningMessage(
				"No messages in queue",
			)
		} else {
			o, _ := json.MarshalIndent(popOutput, "", "  ")
			cli.PrintBoxedInfoMessage(
				fmt.Sprintf(
					"Retrieve message from queue: %s",
					string(o),
				),
			)
		}
		return nil
	},
}
