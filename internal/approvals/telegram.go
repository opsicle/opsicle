package approvals

import "time"

type TelegramRequestSpec struct {
	// AuthorizedResponders is a list of users who are authorized to respond
	// to this request
	AuthorizedResponders AuthorizedResponders `json:"authorizedResponders" yaml:"authorizedResponders"`

	// ChatIds defines the ID of the chat where the message should be sent
	ChatIds []int64 `json:"chatIds" yaml:"chatIds"`

	// Notifications contains details of the messages sent to Telegram
	Notifications Notifications `json:"notifications" yaml:"notifications"`
}

type TelegramResponseSpec struct {
	ChatId     int64     `json:"chatId" yaml:"chatId"`
	Username   string    `json:"username" yaml:"username"`
	UserId     int64     `json:"userId" yaml:"userId"`
	ReceivedAt time.Time `json:"receivedAt" yaml:"receivedAt"`
	Status     Status    `json:"status" yaml:"status"`
}
