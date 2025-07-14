package approvals

import "time"

type TelegramRequestSpec struct {
	// AuthorizedResponders is a list of users who are authorized to respond
	// to this request
	AuthorizedResponders AuthorizedResponders `json:"authorizedResponders" yaml:"authorizedResponders"`

	// ChatId defines the ID of the chat where the message should be sent
	ChatId int64 `json:"chatId" yaml:"chatId"`

	// ChatIds defines the ID of the chat where the message should be sent
	ChatIds []int64 `json:"chatIds" yaml:"chatIds"`

	// MfaSeed is an optional field that when populated, requires the
	// user to respond with their TOTP MFA number. This seed is recommended
	// to be a specially provisioned MFA since you will be sending it to
	// another system
	MfaSeed *string `json:"mfaSeed" yaml:"mfaSeed"`

	// UserId optionally specifies the Telegram user ID of the user who is
	// allowed to approve/reject an approval request
	UserId *int64 `json:"userId" yaml:"userId"`

	// Username optionally specifies the username of the approver whom the
	// approval must come from otherwise the request will be rejected
	Username *string `json:"username" yaml:"username"`

	SentAt *time.Time `json:"sentAt" yaml:"sentAt"`
}

type TelegramResponseSpec struct {
	ChatId     int64     `json:"chatId" yaml:"chatId"`
	Username   string    `json:"username" yaml:"username"`
	UserId     int64     `json:"userId" yaml:"userId"`
	ReceivedAt time.Time `json:"receivedAt" yaml:"receivedAt"`
	Status     Status    `json:"status" yaml:"status"`
}
