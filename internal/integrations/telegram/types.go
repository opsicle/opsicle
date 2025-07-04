package telegram

import (
	"context"

	"github.com/go-telegram/bot/models"
)

// Handler is a wrapper around the third party implementation's handler function
type Handler func(context context.Context, bot *Bot, update *Update)

// Update is a wrapper around the third party implementation's update model
type Update struct {
	CallbackData   string         `json:"callbackData"`
	CallbackId     string         `json:"callbackId"`
	ChatId         int64          `json:"chatId"`
	IsReply        bool           `json:"isReply"`
	Message        string         `json:"message"`
	MessageId      int            `json:"messageId"`
	Raw            *models.Update `json:"-"`
	ReplyMessageId int            `json:"replyMessageId"`
	SenderId       int64          `json:"senderId"`
	SenderUsername string         `json:"senderUsername"`
}
