package approver

import (
	"fmt"
	"opsicle/internal/approver"
	"opsicle/internal/common"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	currentFlag := "redis-enabled"
	Command.PersistentFlags().Bool(
		currentFlag,
		true,
		"when this flag is specified, redis is used as the cache",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "redis-addr"
	Command.PersistentFlags().String(
		currentFlag,
		"localhost:6379",
		"defines the hostname (including port) of the redis server",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "redis-username"
	Command.PersistentFlags().String(
		currentFlag,
		"opsicle",
		"defines the username used to login to redis",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "redis-password"
	Command.PersistentFlags().String(
		currentFlag,
		"password",
		"defines the password used to login to redis",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "telegram-enabled"
	Command.PersistentFlags().Bool(
		currentFlag,
		false,
		"when this flag is specified, the telegram bot is enabled",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "telegram-bot-token"
	Command.PersistentFlags().String(
		currentFlag,
		"",
		"the telegram bot token to be used when telegram is enabled",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)
}

var Command = &cobra.Command{
	Use:     "approver",
	Aliases: []string{"a"},
	Short:   "Starts the approver component",
	Long:    "Starts the approver component which serves as a background job that communicates with the configured component",
	RunE: func(cmd *cobra.Command, args []string) error {

		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)

		isRedisEnabled := viper.GetBool("redis-enabled")
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

		isTelegramEnabled := viper.GetBool("telegram-enabled")
		if isTelegramEnabled {
			telegramBotToken := viper.GetString("telegram-bot-token")
			if telegramBotToken == "" {
				return fmt.Errorf("failed to receive a telegram bot token")
			}
			if err := approver.InitTelegramNotifier(approver.InitTelegramNotifierOpts{
				BotToken: telegramBotToken,
				ChatMap: map[string]int64{
					"main": 267230627,
				},
				ServiceLogs: serviceLogs,
			}); err != nil {
				return fmt.Errorf("failed to initialise telegram client: %s", err)
			}
		}

		if approver.Notifier == nil {
			return fmt.Errorf("failed to identify a notifier")
		}
		go approver.Notifier.StartListening()

		httpServerDone := make(chan common.Done)
		logrus.Infof("starting http client...")
		approver.StartHttpServer(approver.StartHttpServerOpts{
			Addr:        "0.0.0.0:12345",
			Done:        httpServerDone,
			ServiceLogs: serviceLogs,
		})

		return cmd.Help()
	},
}
