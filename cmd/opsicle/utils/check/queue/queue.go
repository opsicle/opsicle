package queue

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/config"
	"opsicle/internal/persistence"
	"opsicle/internal/queue"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{}.
	Append(config.GetNatsFlags())

var Command = cli.NewCommand(cli.CommandOpts{
	Name:    "check.queue",
	Flags:   flags,
	Use:     "queue",
	Aliases: []string{"q"},
	Short:   "Checks queue connectivity",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		appName := opts.GetFullname()
		serviceLogs := opts.GetServiceLogs()

		logrus.Infof("verifying queue connectivity...")
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
		queue.InitNats(queue.InitNatsOpts{
			NatsConnection: natsInstance,
			ServiceLogs:    serviceLogs,
		})
		cli.PrintBoxedSuccessMessage(fmt.Sprintf(
			"Successfully connected to queue at address[%s]",
			viper.GetString("nats-addr"),
		))
		return nil
	},
})
