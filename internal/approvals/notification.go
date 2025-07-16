package approvals

import "time"

type Notifications []Notification

type Notification struct {
	TargetId    string    `json:"targetId"`
	Error       error     `json:"error"`
	IsSuccess   bool      `json:"isSuccess"`
	MessageId   string    `json:"messageId"`
	Platform    string    `json:"platform"`
	RequestUuid string    `json:"requestUuid"`
	SentAt      time.Time `json:"sentAt"`
}
