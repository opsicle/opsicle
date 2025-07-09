package telegrambot

import (
	"context"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/integrations/telegram"

	"github.com/go-telegram/bot/models"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "telegram-bot-token",
		DefaultValue: "",
		Usage:        "the telegram bot token to be used when telegram is enabled",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "telegrambot",
	Aliases: []string{"tgbot", "tg"},
	Short:   "Runs a base telegram bot that returns the chat Id",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		telegramBotToken := viper.GetString("telegram-bot-token")
		if telegramBotToken == "" {
			return fmt.Errorf("failed to receive a telegram bot token")
		}
		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)

		telegramBot, err := telegram.New(telegram.NewOpts{
			BotToken: telegramBotToken,
			DefaultHandler: func(context context.Context, bot *telegram.Bot, update *telegram.Update) {
				serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "received message['%s'] from chat[%v]", update.Message, update.ChatId)

				okButton := &models.InlineKeyboardButton{
					Text:         "Ok",
					CallbackData: "ok",
				}

				markup := &models.InlineKeyboardMarkup{
					InlineKeyboard: [][]models.InlineKeyboardButton{{*okButton}},
				}
				if err := bot.SendMessage(
					update.ChatId,
					fmt.Sprintf("hello, you are in chat `%v`", update.ChatId),
					markup,
				); err != nil {
					logrus.Errorf("failed to send message: %s", err)
				}
			},
			ServiceLogs: serviceLogs,
		})
		if err != nil {
			return fmt.Errorf("failed to create a telegram bot instance: %s", err)
		}
		telegramBot.Start()

		return cmd.Help()
	},
}
