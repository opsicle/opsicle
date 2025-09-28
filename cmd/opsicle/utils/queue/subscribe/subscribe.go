package subscribe

import (
	"context"
	"fmt"
	"math/rand/v2"
	"opsicle/internal/cli"
	"opsicle/internal/common"
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

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "subscribe",
	Aliases: []string{"sub"},
	Short:   "Subscribes to messages from the configured queue",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)

		logrus.Infof("establishing connection to queue...")
		nats, err := queue.InitNats(queue.InitNatsOpts{
			Addr:        viper.GetString("nats-addr"),
			Username:    viper.GetString("nats-username"),
			Password:    viper.GetString("nats-password"),
			NKey:        viper.GetString("nats-nkey-value"),
			ServiceLogs: serviceLogs,
		})
		if err != nil {
			return fmt.Errorf("failed to initialise nats queue: %w", err)
		}
		if err := nats.Connect(); err != nil {
			return fmt.Errorf("failed to connect to nats: %w", err)
		}
		logrus.Infof("established connection to queue")

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
}
