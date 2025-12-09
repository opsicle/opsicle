package pop

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/persistence"
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

var Command = cli.NewCommand(cli.CommandOpts{
	Name:    "utils.queue.pop",
	Flags:   flags,
	Use:     "pop",
	Aliases: []string{"po"},
	Short:   "Pops a message from the configured queue",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		appName := opts.GetFullname()
		serviceLogs := opts.GetServiceLogs()

		logrus.Infof("establishing connection to queue...")
		natsAddr := viper.GetString("nats-addr")
		natsInstance, err := persistence.NewNats(
			persistence.NatsConnectionOpts{
				AppName: appName,
				Host:    natsAddr,
			},
			persistence.NatsAuthOpts{
				NKey: viper.GetString("nats-nkey-value"),
			},
			&serviceLogs,
		)
		if err != nil {
			return fmt.Errorf("failed to create nats client: %w", err)
		}
		if err := natsInstance.Init(); err != nil {
			return fmt.Errorf("failed to connect to nats: %w", err)
		}
		opts.AddShutdownProcess("nats", natsInstance.Shutdown)
		queue.InitNats(queue.InitNatsOpts{
			NatsConnection: natsInstance,
			ServiceLogs:    serviceLogs,
		})
		logrus.Infof("established connection to queue")
		nats := queue.Get()

		logrus.Infof("popping message from queue using consumer[%s]...", appName)
		popOutput, err := nats.Pop(queue.PopOpts{
			ConsumerId: opts.GetSnakeCaseName(),
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
			logrus.Infof("received message from queue:\n%s", string(o))
			cli.PrintBoxedInfoMessage(
				fmt.Sprintf(
					"Retrieve message from queue: %s",
					string(popOutput.Data),
				),
			)
		}
		return nil
	},
})
