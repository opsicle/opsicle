package approver

import (
	"fmt"
	"opsicle/internal/approver"
	"opsicle/internal/cli"
	"opsicle/internal/common"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	flags.AddToCommand(Command)
}

var flags cli.Flags = cli.Flags{
	{
		Name:         "redis-enabled",
		DefaultValue: true,
		Usage:        "when this flag is specified, redis is used as the cache",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "redis-addr",
		DefaultValue: "localhost:6379",
		Usage:        "defines the hostname (including port) of the redis server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "redis-username",
		DefaultValue: "opsicle",
		Usage:        "defines the username used to login to redis",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "redis-password",
		DefaultValue: "password",
		Usage:        "defines the password used to login to redis",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "slack-enabled",
		DefaultValue: false,
		Usage:        "when this flag is specified, the slack bot is enabled",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "slack-app-token",
		DefaultValue: "",
		Usage:        "the slack app token to be used when slack is enabled",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "slack-bot-token",
		DefaultValue: "",
		Usage:        "the slack bot token to be used when slack is enabled",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "telegram-enabled",
		DefaultValue: false,
		Usage:        "when this flag is specified, the telegram bot is enabled",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "telegram-bot-token",
		DefaultValue: "",
		Usage:        "the telegram bot token to be used when telegram is enabled",
		Type:         cli.FlagTypeString,
	},
}

var Command = &cobra.Command{
	Use:     "approver",
	Aliases: []string{"a"},
	Short:   "Starts the approver component",
	Long:    "Starts the approver component which serves as a background job that communicates with the configured component",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)

		isRedisEnabled := viper.GetBool("redis-enabled")
		logrus.Debugf("redis-enabled status: %v", isRedisEnabled)
		if isRedisEnabled {
			if err := approver.InitRedisCache(approver.InitRedisCacheOpts{
				Addr:        viper.GetString("redis-addr"),
				Username:    viper.GetString("redis-username"),
				Password:    viper.GetString("redis-password"),
				ServiceLogs: serviceLogs,
			}); err != nil {
				return fmt.Errorf("failed to initialise redis cache: %s", err)
			}
			logrus.Infof("redis client initialised")
		}

		isSlackEnabled := viper.GetBool("slack-enabled")
		logrus.Debugf("slack-enabled status: %v", isSlackEnabled)
		if isSlackEnabled {
			slackBotToken := viper.GetString("slack-bot-token")
			if slackBotToken == "" {
				return fmt.Errorf("failed to receive a slack bot token")
			}
			slackAppToken := viper.GetString("slack-app-token")
			if slackAppToken == "" {
				return fmt.Errorf("failed to receive a slack app token")
			}
			approver.InitSlackNotifier(approver.InitSlackNotifierOpts{
				AppToken:    slackAppToken,
				BotToken:    slackBotToken,
				ServiceLogs: serviceLogs,
			})
			logrus.Infof("slack notifier initialised")
		}

		isTelegramEnabled := viper.GetBool("telegram-enabled")
		logrus.Debugf("telegram-enabled status: %v", isTelegramEnabled)
		if isTelegramEnabled {
			telegramBotToken := viper.GetString("telegram-bot-token")
			if telegramBotToken == "" {
				return fmt.Errorf("failed to receive a telegram bot token")
			}
			if err := approver.InitTelegramNotifier(approver.InitTelegramNotifierOpts{
				BotToken:    telegramBotToken,
				ServiceLogs: serviceLogs,
			}); err != nil {
				return fmt.Errorf("failed to initialise telegram client: %s", err)
			}
			logrus.Infof("telegram notifier initialised")
		}

		logrus.Debugf("verifying notifiers...")
		if approver.Notifiers == nil {
			return fmt.Errorf("failed to identify a notifier")
		}
		logrus.Debugf("starting notifiers...")
		go approver.Notifiers.StartListening()

		logrus.Debugf("starting http server...")
		httpServerDone := make(chan common.Done)
		approver.StartHttpServer(approver.StartHttpServerOpts{
			Addr:        "0.0.0.0:12345",
			Done:        httpServerDone,
			ServiceLogs: serviceLogs,
		})

		return cmd.Help()
	},
}
