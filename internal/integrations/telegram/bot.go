package telegram

import (
	"context"
	"fmt"
	"opsicle/internal/common"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Bot represents a Telegram bot instance
type Bot struct {
	// Client is an instance of the third-party library we use for
	// interacting with Telegram
	Client *bot.Bot

	// Done is a channel that upon receiving a message, terminates
	// the bot gracefully
	Done chan common.Done

	// ServiceLogs is the channel to send logs to for logging via
	// the centralised logger
	ServiceLogs chan<- common.ServiceLog

	// SubHandlers are
	SubHandlers []Handler

	Raw *bot.Bot
}

func (b *Bot) UpdateMessage(chatId int64, messageId int, newMessage string, markup ...*models.ReplyMarkup) error {
	b.ServiceLogs <- common.ServiceLogf(
		common.LogLevelDebug,
		"chat[%v].updateMessage[%v] '%s' (markup: %v)",
		chatId,
		messageId,
		newMessage,
		len(markup) > 0,
	)
	editMessageParameters := &bot.EditMessageTextParams{
		ChatID:    chatId,
		MessageID: messageId,
		ParseMode: "MarkdownV2",
		Text:      newMessage,
	}
	if markup[0] == nil {
		editMessageParameters.ReplyMarkup = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{}}
	} else {
		editMessageParameters.ReplyMarkup = markup[0]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := b.Client.EditMessageText(ctx, editMessageParameters); err != nil {
		return fmt.Errorf("failed to edit text of message[%v] in chat[%v]: %s", messageId, chatId, err)
	}
	return nil
}

func (b *Bot) ReplyMessage(chatId int64, replyMessageId int, message string, markup ...models.ReplyMarkup) error {
	b.ServiceLogs <- common.ServiceLogf(
		common.LogLevelDebug,
		"chat[%v] >> '%s'", chatId, message,
	)
	messageParameters := &bot.SendMessageParams{
		ChatID: chatId,
		Text:   message,
		ReplyParameters: &models.ReplyParameters{
			ChatID:    chatId,
			MessageID: replyMessageId,
		},
		ParseMode: "MarkdownV2",
	}
	if len(markup) > 0 {
		messageParameters.ReplyMarkup = markup[0]
	}
	ctx := context.Background()
	if _, err := b.Client.SendMessage(ctx, messageParameters); err != nil {
		return fmt.Errorf("failed to send message: %s", err)
	}
	return nil
}

func (b *Bot) SendMessage(chatId int64, message string, markup ...models.ReplyMarkup) error {
	b.ServiceLogs <- common.ServiceLogf(
		common.LogLevelDebug,
		"chat[%v] >> '%s'", chatId, message,
	)
	messageParameters := &bot.SendMessageParams{
		ChatID:    chatId,
		Text:      message,
		ParseMode: "MarkdownV2",
	}
	if len(markup) > 0 {
		messageParameters.ReplyMarkup = markup[0]
	}
	ctx := context.Background()
	if _, err := b.Client.SendMessage(ctx, messageParameters); err != nil {
		return fmt.Errorf("failed to send message: %s", err)
	}
	return nil
}

func (b *Bot) Start() {
	go func() {
		<-b.Done
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := b.Client.Close(ctx); err != nil {
			b.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to close bot: %s", err)
		}
	}()
	b.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "starting a telegram bot...")
	b.Client.Start(context.Background())
}
