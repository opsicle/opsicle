package approvals

import (
	"opsicle/internal/common"
	"os"

	"gopkg.in/yaml.v3"
)

type Request struct {
	common.Resource `json:"resource" yaml:",inline"`
	Spec            RequestSpec `json:"spec" yaml:"spec"`
}

type RequestSpec struct {
	Id            string                `json:"id" yaml:"id"`
	Message       string                `json:"message" yaml:"message"`
	RequesterId   string                `json:"requesterId" yaml:"requesterId"`
	RequesterName string                `json:"requesterName" yaml:"requesterName"`
	Telegram      []TelegramRequestSpec `json:"telegram" yaml:"telegram"`
	Type          string                `json:"type" yaml:"type"`
}

type TelegramRequestSpec struct {
	ChatId int64 `json:"chatId" yaml:"chatId"`
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
	return &approvalRequest, nil
}
