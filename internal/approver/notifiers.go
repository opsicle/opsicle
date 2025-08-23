package approver

import (
	"fmt"
	"time"
)

const (
	NotifierPlatformSlack    = "slack"
	NotifierPlatformTelegram = "telegram"
)

var Notifiers notifiers

type notifiers []notifier

func (n notifiers) SendApprovalRequest(req *ApprovalRequest) (requestUuid string, notifications []notificationMessage, err error) {
	for _, notifierInstance := range n {
		var notifierInstanceNotifications []notificationMessage
		requestUuid, notifierInstanceNotifications, err = notifierInstance.SendApprovalRequest(req)
		if err != nil {
			return "", nil, fmt.Errorf("failed to send all approval requests: %w", err)
		}
		notifications = append(notifications, notifierInstanceNotifications...)
	}
	return requestUuid, notifications, err
}

func (n notifiers) StartListening() {
	for _, notifierInstance := range n {
		notifierInstance.StartListening()
	}
}

func (n notifiers) Stop() {
	for _, notifierInstance := range n {
		notifierInstance.Stop()
	}
}

type notifier interface {
	SendApprovalRequest(req *ApprovalRequest) (requestUuid string, notifications notificationMessages, err error)
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
