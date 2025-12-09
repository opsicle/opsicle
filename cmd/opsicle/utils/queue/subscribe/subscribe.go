package subscribe

import (
	"context"
	"fmt"
	"math/rand/v2"
	"opsicle/internal/cli"
	"opsicle/internal/persistence"
	"opsicle/internal/queue"
	"time"

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
	Name:    "utils.queue.subscribe",
	Flags:   flags,
	Use:     "subscribe",
	Aliases: []string{"sub"},
	Short:   "Subscribes to messages from the configured queue",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		appName := opts.GetFullname()
		serviceLogs := opts.GetServiceLogs()

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
		opts.AddShutdownProcess("nats", natsInstance.Shutdown)
		logrus.Infof("established connection to queue")
		nats := queue.Get()

		logrus.Infof("subscribing to queue...")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := nats.Subscribe(queue.SubscribeOpts{
			NakBackoff: 3 * time.Second,
			ConsumerId: "utils-queue-subscribe",
			Context:    ctx,
			Handler: func(ctx context.Context, nm queue.Message) error {
				logrus.Infof("⚙️  processing message: %s", string(nm.Data))
				if rand.Float32() < 0.3 {
					return nil
				}
				if rand.Float32() < 0.6 {
					processingTime := time.Duration(int(rand.Float32()*10)%4) * time.Second
					logrus.Infof("⏳ simulating delay of duration[%v]", processingTime)
					<-time.After(processingTime)
					return nil
				}
				return fmt.Errorf("🔥 simulated error on message[%s]", string(nm.Data))
			},
			Queue: queue.QueueOpts{
				Subject: "test",
				Stream:  "utils_queue",
			},
		}); err != nil {
			return fmt.Errorf("failed to subscribe: %w", err)
		}
		return nil
	},
})
