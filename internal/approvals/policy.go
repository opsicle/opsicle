package approvals

import "opsicle/internal/common"

type Policy struct {
	common.Resource `json:"resource" yaml:",inline"`
	Spec            PolicySpec `json:"spec" yaml:"spec"`
}

type PolicySpec struct {
	// Id is a user-provided string that identifies this policy and
	// has to be unique
	Id string `json:"id" yaml:"id"`

	// Uuid is generated on initialisation
	Uuid *string `json:"uuid" yaml:"uuid"`

	// Namespace is a multi-purpose string that can be used for things
	// like multi-tenancy and allowing the policy to be owned by a owning
	// entity
	Namespace *string `json:"namespace" yaml:"namespace"`

	// Slack specifies target channels and authorised responders on the
	// Slack communication platform
	Slack SlackRequestSpec `json:"slack" yaml:"slack"`

	// Telegram specifies target chats and authorised responders on the
	// Telegram communication platform
	Telegram TelegramRequestSpec `json:"telegram" yaml:"telegram"`
}
