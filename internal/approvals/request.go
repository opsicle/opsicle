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

	// AuthorizedApprovers specifies which users are allowed to
	// perform an approval/rejection
	AuthorizedApprovers *AuthorizedApproversSpec `json:"authorizedApprovers" yaml:"authorizedApprovers"`

	// Id is the ID of a request which will be the same for all
	// requests of a given type
	Id string `json:"id" yaml:"id"`

	// Uuid is populated by the approver service if it's not already
	// defined and represents the ID of the request instance (instead of
	// the request)
	Uuid *string `json:"uuid" yaml:"uuid"`

	// Message is an additiona message describing the request
	Message string `json:"message" yaml:"message"`

	// MfaSeed is an optional field that when populated, requires the
	// user to respond with their TOTP MFA number. This seed is recommended
	// to be a specially provisioned MFA since you will be sending it to
	// another system
	MfaSeed *string `json:"mfaSeed" yaml:"mfaSeed"`

	// RequesterName indicates the requester's system ID
	RequesterId string `json:"requesterId" yaml:"requesterId"`

	// RequesterName indicates the requester's name
	RequesterName string `json:"requesterName" yaml:"requesterName"`

	// Telegram specifies the targets in Telegram to send this request to
	Telegram []TelegramRequestSpec `json:"telegram" yaml:"telegram"`

	// TtlSeconds indicates the duration in seconds until the request expires
	TtlSeconds int `json:"ttlSeconds" yaml:"ttlSeconds"`
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

type AuthorizedApproversSpec struct {
	Telegram AuthorizedTelegramApprovers `json:"telegram" yaml:"telegram"`
}

type AuthorizedTelegramApprovers []AuthorizedTelegramApprovers
type AuthorizedTelegramApprover struct {
	// UserId specifies the Telegram user ID of the user who is allowed
	// to approve/reject an approval request
	UserId int64 `json:"userId" yaml:"userId"`

	// ChatId optionally specifies the ID of the chat which the approval
	// must come from otherwise the request will be rejected
	ChatId *int64 `json:"chatId" yaml:"chatId"`

	// Username optionally specifies the username of the approver whom the
	// approval must come from otherwise the request will be rejected
	Username *string `json:"username" yaml:"username"`
}

type TelegramRequestSpec struct {
	ChatId int64      `json:"chatId" yaml:"chatId"`
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
