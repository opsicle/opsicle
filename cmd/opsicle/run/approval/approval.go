package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"opsicle/internal/approvals"
	"opsicle/internal/approver"
	"opsicle/internal/common"
	"opsicle/internal/integrations/telegram"
	"os"
	"sync"
	"time"

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

		approvalRequest := approver.ApprovalRequest{
			Spec: approvalRequestInstance.Spec,
		}

		var waiter sync.WaitGroup

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
		if isTelegramEnabled {
			logrus.Infof("initialising telegram notifier...")
			telegramBotToken := viper.GetString("telegram-bot-token")
			if telegramBotToken == "" {
				return fmt.Errorf("failed to receive a telegram bot token")
			}
			isDone := false
			if err := approver.InitTelegramNotifier(approver.InitTelegramNotifierOpts{
				BotToken: telegramBotToken,
				DefaultHandler: func(context context.Context, bot *telegram.Bot, update *telegram.Update) {
					// at the end of this, terminate the function
					defer func() {
						if isDone {
							waiter.Done()
						}
					}()

					// if it's not an approve/reject button call, do nothing
					if update.CallbackData == "" {
						return
					}
					action, notificationId, requestId, err := approver.ParseTelegramApprovalCallbackData(update.CallbackData)
					if err != nil {
						logrus.Errorf("failed to parse callback: %s", err)
						if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
							logrus.Errorf("failed to send error response to user: %s", err)
						}
						isDone = true
						return
					}
					logrus.Infof("received callback with action[%s] on request[%s:%s]", action, notificationId, requestId)
					currentApprovalRequest, err := approver.LoadApprovalRequest(approver.ApprovalRequest{
						Spec: approvals.RequestSpec{
							Id:   requestId,
							Uuid: &notificationId,
						},
					})
					if err != nil {
						logrus.Errorf("failed to fetch request from cache: %v", err)
						if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
							logrus.Errorf("failed to send error response to user: %s", err)
						}
						isDone = true
						return
					}
					o, _ := json.MarshalIndent(currentApprovalRequest, "", "  ")
					logrus.Infof("retrieved approval request:\n%s", string(o))

					var response = ""
					if currentApprovalRequest.Spec.Approval != nil {
						if currentApprovalRequest.Spec.Approval.Status == "approved" {
							response = "This request has already been approved"
						} else {
							response = "This request has already been rejected"
						}
					} else if action == approver.ActionApprove {
						if err := bot.UpdateMessage(
							update.ChatId,
							update.MessageId,
							fmt.Sprintf(
								"✅ Approval request\nID: `%v`\nMessage: `%s`\nRequester: %s \\(`%s`\\)\n\nStatus: **APPROVED**",
								currentApprovalRequest.Spec.Id,
								currentApprovalRequest.Spec.Message,
								currentApprovalRequest.Spec.RequesterName,
								currentApprovalRequest.Spec.RequesterId,
							),
							nil,
						); err != nil {
							logrus.Errorf("failed to update message[%v]: %s", update.MessageId, err)
						}
					} else if action == approver.ActionReject {
						if err := bot.UpdateMessage(
							update.ChatId,
							update.MessageId,
							fmt.Sprintf(
								"❌ Approval request\nID: `%v`\nMessage: `%s`\nRequester: %s \\(`%s`\\)\n\nStatus: **REJECTED**",
								currentApprovalRequest.Spec.Id,
								currentApprovalRequest.Spec.Message,
								currentApprovalRequest.Spec.RequesterName,
								currentApprovalRequest.Spec.RequesterId,
							),
							nil,
						); err != nil {
							logrus.Errorf("failed to update message[%v]: %s", update.MessageId, err)
						}
					}
					if response != "" {
						if err := bot.ReplyMessage(update.ChatId, update.MessageId, response); err != nil {
							serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send confirmation message: %v", err)
						}
					}
					<-time.After(1 * time.Second)
					isDone = true
				},
				ServiceLogs: serviceLogs,
			}); err != nil {
				return fmt.Errorf("failed to initialise telegram client: %s", err)
			}
			logrus.Infof("starting telegram client...")
			waiter.Add(1)
			go approver.Notifier.StartListening()
		}

		logrus.Infof("dispatching approval request messages...")
		notificationId, sentApprovals, err := approver.Notifier.SendApprovalRequest(approvalRequest)
		if err != nil {
			return fmt.Errorf("failed to send an approval request message: %s", err)
		}
		approvalRequest.Spec.NotificationId = &notificationId
		logrus.Infof("updating approval request in the cache...")
		if err := approver.CreateApprovalRequest(approvalRequest); err != nil {
			return fmt.Errorf("failed to create the approval request: %s", err)
		}
		logrus.Infof("approval requested updated in the cache")

		for _, sentApproval := range sentApprovals {
			if sentApproval.IsSuccess {
				logrus.Infof(
					"sent approvalRequest[%s] to platform[%s] on chat[%v] as message[%s]",
					sentApproval.Id,
					sentApproval.Platform,
					sentApproval.TargetId,
					sentApproval.MessageId,
				)
			} else {
				logrus.Errorf(
					"approvalRequest[%s] did not get sent: %s",
					sentApproval.Id,
					sentApproval.Error,
				)
			}
		}

		waiter.Wait()

		return cmd.Help()
	},
}
