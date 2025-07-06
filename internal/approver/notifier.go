package approver

import (
	"time"
)

const (
	NotifierPlatformTelegram = "telegram"
)

var Notifier notifier

type notifier interface {
	SendApprovalRequest(req ApprovalRequest) (notificationId string, notifications []notificationMessage, err error)
	StartListening()
	Stop()
}

type notificationMessages []notificationMessage
type notificationMessage struct {
	// Error when defined indiciates that the message failed to be sent
	Error error

	// Id is a uniquely generated ID for every message
	Id string

	// IsSuccess defines whether the message was sent successfully
	IsSuccess bool

	// MessageId is the in-platform ID of the message - this
	// should be usable to update the message by the
	// platform's SDK
	MessageId string

	// Platform is a string indicating which platform this
	// message was sent to
	Platform notificationPlatform

	// SentAt is the timestamp where the notification message
	// was sent
	SentAt time.Time

	// TargetId is the in-platform ID of the channel/group/chat
	// that the message was sent to
	TargetId string
}

type notificationPlatform string
