package approvals

import "time"

type SlackRequestSpec struct {
	// AuthorizedResponders is a list of users who are authorized to respond
	// to this request
	AuthorizedResponders AuthorizedResponders `json:"authorizedResponders" yaml:"authorizedResponders"`

	// ChannelNames defines the name of the channel to send to
	ChannelNames []string `json:"channelNames" yaml:"channelNames"`

	// ChannelId defines the ID of the channel to send to, if this
	// is populated, this will be used, otherwise this will be
	// populated when the `.ChannelName` is evaluated
	ChannelIds []string `json:"channelIds" yaml:"channelIds"`

	// Notifications contains details of the messages sent to Slack
	Notifications Notifications `json:"notifications" yaml:"notifications"`
}

type SlackResponseSpec struct {
	ChannelId  string    `json:"channelId" yaml:"channelId"`
	ReceivedAt time.Time `json:"receivedAt" yaml:"receivedAt"`
	Status     Status    `json:"status" yaml:"status"`
	UserId     string    `json:"userId" yaml:"userId"`
	UserName   string    `json:"userName" yaml:"userName"`
}
