package approvals

import (
	"opsicle/internal/common"
	"os"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type Request struct {
	common.Resource `json:"resource" yaml:",inline"`
	Spec            RequestSpec `json:"spec" yaml:"spec"`
}

type RequestSpec struct {
	// Actions is the audit log trail of any actions taken on the ApprovalRequest
	// that this request specification is attached to
	Actions Actions `json:"actions" yaml:"actions"`

	// Approval is populated after an approval/rejection action
	// happens
	Approval *ApprovalSpec `json:"approval" yaml:"approval"`

	// Callback is a field that when specified, results in the approver
	// service processing a callback to the specified endpoint
	Callback *CallbackSpec `json:"callback" yaml:"callback"`

	// Id is the ID of a request which will be the same for all
	// requests of a given type
	Id string `json:"id" yaml:"id"`

	// Links are optional additional links to view the request in a browser or
	// other application,
	Links []RequestLinkAttachment `json:"links" yaml:"links"`

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

	// Uuid is populated by the approver service if it's not already
	// defined and represents the ID of the request instance (instead of
	// the request)
	Uuid *string `json:"uuid" yaml:"uuid"`
}

type RequestLinkAttachment struct {
	Url         string `json:"url" yaml:"url"`
	Description string `json:"description" yaml:"description"`
}

type CallbackSpec struct {
	Webhook *WebhookCallbackSpec `json:"webhook" yaml:"webhook"`

	// Type defines the type of this callback. If this is not specified,
	// the precedence will follow the code tagged with #callback-type-priority
	Type CallbackType `json:"type" yaml:"type"`
}

type WebhookCallbackSpec struct {
	// Method defines the HTTP method used when sending the call
	Method string `json:"method" yaml:"method"`

	// Url specifies the URL that will be called with the full approval
	// request details when a request is approved/rejected
	Url string `json:"url" yaml:"url"`

	// RetryCount specifies the number of retries that will be made
	// if the callback fails. Defaults to 5 with an exponential backoff
	// strategy unless otherwise specfied
	RetryCount *int `json:"retryCount" yaml:"retryCount"`

	// RetryIntervalSeconds when specified, forces the retry mechanism
	// to perform retries at fxed intervals
	RetryIntervalSeconds *int `json:"retryIntervalSeconds" yaml:"retryIntervalSeconds"`

	// Auth defines the auth mechanism to use when authenticating with
	// the specified `.Url`
	Auth *WebhookCallbackAuthSpec `json:"auth" yaml:"auth"`
}

type WebhookCallbackAuthSpec struct {
	Basic  *WebhookCallbackBasicAuthSpec  `json:"basic" yaml:"basic"`
	Bearer *WebhookCallbackBearerAuthSpec `json:"bearer" yaml:"bearer"`
	Header *WebhookCallbackHeaderAuthSpec `json:"header" yaml:"header"`
}

type WebhookCallbackBasicAuthSpec struct {
	Password string `json:"password" yaml:"password"`
	Username string `json:"username" yaml:"username"`
}

type WebhookCallbackBearerAuthSpec struct {
	Value string `json:"value" yaml:"value"`
}

type WebhookCallbackHeaderAuthSpec struct {
	Key   string `json:"key" yaml:"key"`
	Value string `json:"value" yaml:"value"`
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
