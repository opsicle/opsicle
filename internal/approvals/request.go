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
	Approval       *ApprovalSpec         `json:"approval" yaml:"approval"`
	Id             string                `json:"id" yaml:"id"`
	Uuid           *string               `json:"uuid" yaml:"uuid"`
	Message        string                `json:"message" yaml:"message"`
	NotificationId *string               `json:"notificationId" yaml:"notificationId"`
	RequesterId    string                `json:"requesterId" yaml:"requesterId"`
	RequesterName  string                `json:"requesterName" yaml:"requesterName"`
	Telegram       []TelegramRequestSpec `json:"telegram" yaml:"telegram"`
	Type           string                `json:"type" yaml:"type"`
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
