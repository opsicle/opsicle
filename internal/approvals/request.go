package approvals

import (
	"opsicle/internal/common"
	"os"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type Request struct {
	common.Resource `json:"resource" yaml:",inline"`
	Spec            RequestSpec `json:"spec" yaml:"spec"`
}

type RequestSpec struct {
	// Approval is populated after an approval/rejection action
	// happens
	Approval *ApprovalSpec `json:"approval" yaml:"approval"`

	// Id is the ID of a request which will be the same for all
	// requests of a given type
	Id string `json:"id" yaml:"id"`

	// Uuid is populated by the approver service if it's not already
	// defined and represents the ID of the request instance (instead of
	// the request)
	Uuid *string `json:"uuid" yaml:"uuid"`

	// Message is an additional message describing the request
	Message string `json:"message" yaml:"message"`

	// RequesterName indicates the requester's system ID
	RequesterId string `json:"requesterId" yaml:"requesterId"`

	// RequesterName indicates the requester's name
	RequesterName string `json:"requesterName" yaml:"requesterName"`

	// Slack specifies the targets in Slack to send this request to
	Slack []SlackRequestSpec `json:"slack" yaml:"slack"`

	// Telegram specifies the targets in Telegram to send this request to
	Telegram []TelegramRequestSpec `json:"telegram" yaml:"telegram"`

	// Title is an optional field that when sepcified, is used as the first
	// line of the approval request message
	Title *string `json:"title" yaml:"title"`

	// TtlSeconds indicates the duration in seconds until the request expires
	TtlSeconds int `json:"ttlSeconds" yaml:"ttlSeconds"`

	// Url is an optional additional link to view the request in a browser or
	// other application,
	Url *string `json:"url" yaml:"url"`

	// WebhookUrl is an optional field that when specified, will be called with
	// a HTTP POST request with the full approval request details when a request
	// is approved/rejected
	WebhookUrl *string `json:"webhookUrl" yaml:"webhookUrl"`
}

func (rs *RequestSpec) Init() {
	if rs.Uuid != nil {
		return
	}
	randomUuid := uuid.New().String()
	rs.Uuid = &randomUuid
}

func (rs *RequestSpec) GetUuid() string {
	if rs.Uuid == nil {
		rs.Init()
	}
	return *rs.Uuid
}

type SlackRequestSpec struct {
	// ChannelName defines the name of the channel to send to
	ChannelName string `json:"channelName" yaml:"channelName"`

	// ChannelId defines the ID of the channel to send to, if this
	// is populated, this will be used, otherwise this will be
	// populated when the `.ChannelName` is evaluated
	ChannelId *string `json:"channelId" yaml:"channelId"`

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

type TelegramRequestSpec struct {
	// ChatId defines the ID of the chat where the message should be sent
	ChatId int64 `json:"chatId" yaml:"chatId"`

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

// LoadRequestFromFile reads YAML from file and returns an Automation
func LoadRequestFromFile(path string) (*Request, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var approvalRequest Request
	if err := yaml.Unmarshal(data, &approvalRequest); err != nil {
		return nil, err
	}
	approvalRequest.Spec.Init()
	return &approvalRequest, nil
}
