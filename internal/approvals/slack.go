package approvals

import "time"

type SlackRequestSpec struct {
	// AuthorizedResponders is a list of users who are authorized to respond
	// to this request
	AuthorizedResponders AuthorizedResponders `json:"authorizedResponders" yaml:"authorizedResponders"`

	// ChannelName defines the name of the channel to send to
	ChannelName string `json:"channelName" yaml:"channelName"`

	// ChannelNames defines the name of the channel to send to
	ChannelNames []string `json:"channelNames" yaml:"channelNames"`

	// ChannelId defines the ID of the channel to send to, if this
	// is populated, this will be used, otherwise this will be
	// populated when the `.ChannelName` is evaluated
	ChannelId *string `json:"channelId" yaml:"channelId"`

	// ChannelId defines the ID of the channel to send to, if this
	// is populated, this will be used, otherwise this will be
	// populated when the `.ChannelName` is evaluated
	ChannelIds []string `json:"channelIds" yaml:"channelIds"`

	// MessageId is not meant to be specified in the approval request
	// manifest, it is populated after the approval request message
	// is sent so that responses can be threaded
	MessageId *string `json:"-" yaml:"-"`

	// MfaSeed is an optional field that when populated, requires the
	// user to respond with their TOTP MFA number. This seed is recommended
	// to be a specially provisioned MFA since you will be sending it to
	// another system
	MfaSeed *string `json:"mfaSeed" yaml:"mfaSeed"`

	// UserId optionally specifies the Slack user ID of the user who is
	// allowed to approve/reject an approval request
	UserId *string `json:"userId" yaml:"userId"`

	SentAt *time.Time `json:"sentAt" yaml:"sentAt"`
}

type SlackResponseSpec struct {
	ChannelId  string    `json:"channelId" yaml:"channelId"`
	UserId     string    `json:"userId" yaml:"userId"`
	UserName   string    `json:"userName" yaml:"userName"`
	ReceivedAt time.Time `json:"receivedAt" yaml:"receivedAt"`
	Status     Status    `json:"status" yaml:"status"`
}
