package push

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/queue"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "count",
		DefaultValue: 0,
		Usage:        "Specifies the number of messages to post, the number will be appended to the input message",
		Type:         cli.FlagTypeInteger,
	},
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
	Use:     "push <message>",
	Aliases: []string{"pu"},
	Short:   "Pushes the specified <message> to the configured queue",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			cli.PrintBoxedErrorMessage(
				"The <message> parameter is required when invoking this command",
			)
			return fmt.Errorf("failed to receive input message")
		}
		message := strings.Join(args, " ")
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

		messageCount := viper.GetInt("count")
		queueOpts := queue.QueueOpts{
			Subject: "test",
			Stream:  "utils_queue",
		}

		if messageCount > 1 {
			logrus.Infof("pushing %v messages to queue...", messageCount)
			for i := range messageCount {
				currentMessage := fmt.Sprintf("%s %v", message, i)
				_, err := nats.Push(queue.PushOpts{
					Data:  []byte(currentMessage),
					Queue: queueOpts,
				})
				if err != nil {
					return fmt.Errorf("failed to push message: %w", err)
				}
			}
			cli.PrintBoxedSuccessMessage(
				fmt.Sprintf(
					"pushed %v messages with base text '%s' to stream[%s] and subject[%s]",
					messageCount,
					message,
					"test",
					"utils_queue",
				),
			)
			return nil
		}
		logrus.Infof("pushing message to queue...")
		pushOutput, err := nats.Push(queue.PushOpts{
			Data:  []byte(message),
			Queue: queueOpts,
		})
		if err != nil {
			return fmt.Errorf("failed to push message: %w", err)
		}
		cli.PrintBoxedSuccessMessage(
			fmt.Sprintf(
				"pushed message of size[%v] to stream[%s] and subject[%s]",
				pushOutput.MessageSizeBytes,
				pushOutput.Queue.Stream,
				pushOutput.Queue.Subject,
			),
		)

		return nil
	},
}
