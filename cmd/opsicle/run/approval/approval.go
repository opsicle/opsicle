package approval

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/approvals"
	"opsicle/internal/approver"
	"opsicle/internal/common"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const cmdCtx = "o-run-approval-"

func init() {
	currentFlag := "redis-enabled"
	Command.Flags().Bool(
		currentFlag,
		true,
		"when this flag is specified, redis is used as the cache",
	)
	viper.BindPFlag(cmdCtx+currentFlag, Command.Flags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "redis-addr"
	Command.Flags().String(
		currentFlag,
		"localhost:6379",
		"defines the hostname (including port) of the redis server",
	)
	viper.BindPFlag(cmdCtx+currentFlag, Command.Flags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "redis-username"
	Command.Flags().String(
		currentFlag,
		"opsicle",
		"defines the username used to login to redis",
	)
	viper.BindPFlag(cmdCtx+currentFlag, Command.Flags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "redis-password"
	Command.Flags().String(
		currentFlag,
		"password",
		"defines the password used to login to redis",
	)
	viper.BindPFlag(cmdCtx+currentFlag, Command.Flags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "telegram-enabled"
	Command.Flags().Bool(
		currentFlag,
		false,
		"when this flag is specified, the telegram bot is enabled",
	)
	viper.BindPFlag(cmdCtx+currentFlag, Command.Flags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "telegram-bot-token"
	Command.Flags().String(
		currentFlag,
		"",
		"the telegram bot token to be used when telegram is enabled",
	)
	viper.BindPFlag(cmdCtx+currentFlag, Command.Flags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)
}

var Command = &cobra.Command{
	Use:   "approval",
	Short: "Runs an approval manifest",
	RunE: func(cmd *cobra.Command, args []string) error {
		resourceIsSpecified := false
		resourcePath := ""
		if len(args) > 0 {
			resourcePath = args[0]
			resourceIsSpecified = true
		}
		if !resourceIsSpecified {
			return fmt.Errorf("failed to receive a <path-to-template-file")
		}
		fi, err := os.Stat(resourcePath)
		if err != nil {
			return fmt.Errorf("failed to check for existence of file at path[%s]: %s", resourcePath, err)
		}
		if fi.IsDir() {
			return fmt.Errorf("failed to get a file at path[%s]: got a directory", resourcePath)
		}
		approvalRequestInstance, err := approvals.LoadRequestFromFile(resourcePath)
		if err != nil {
			return fmt.Errorf("failed to load approval request: %s", err)
		}
		o, _ := json.MarshalIndent(approvalRequestInstance, "", "  ")
		logrus.Infof("loaded approval request as follows:\n%s", string(o))
		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)

		approvalRequest := approver.ApprovalRequest(approvalRequestInstance.Spec)

		// initialise the caches

		logrus.Infof("initialising caches...")
		isRedisEnabled := viper.GetBool(cmdCtx + "redis-enabled")
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

		logrus.Infof("creating approval request in the cache...")
		if err := approver.CreateApprovalRequest(approvalRequest); err != nil {
			return fmt.Errorf("failed to create the approval request: %s", err)
		}
		logrus.Infof("approval requested created in the cache")

		// initialise the notifiers

		logrus.Infof("initialising notifiers...")
		isTelegramEnabled := viper.GetBool(cmdCtx + "telegram-enabled")
		telegramDone := make(chan common.Done)
		if isTelegramEnabled {
			logrus.Infof("initialising telegram notifier...")
			telegramBotToken := viper.GetString("telegram-bot-token")
			if telegramBotToken == "" {
				return fmt.Errorf("failed to receive a telegram bot token")
			}
			if err := approver.InitTelegramApprover(approver.InitTelegramApproverOpts{
				BotToken:    telegramBotToken,
				ServiceLogs: serviceLogs,
			}); err != nil {
				return fmt.Errorf("failed to initialise telegram client: %s", err)
			}
			logrus.Infof("sending approval message...")
			if err := approver.TelegramApprover.SendApproval(approvalRequest); err != nil {
				return fmt.Errorf("failed to send an approval request message: %s", err)
			}
			logrus.Infof("starting telegram client...")
			approver.TelegramApprover.Start(telegramDone)
		}

		return cmd.Help()
	},
}
