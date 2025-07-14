package approvals

import (
	"opsicle/internal/common"
	"time"
)

type Approval struct {
	common.Resource `json:"resource" yaml:",inline"`
	Spec            ApprovalSpec `json:"spec" yaml:"spec"`
}

type ApprovalSpec struct {
	ApproverId      string                 `json:"approverId" yaml:"approverId"`
	ApproverName    string                 `json:"approverName" yaml:"approverName"`
	Id              string                 `json:"id" yaml:"id"`
	RequestId       string                 `json:"requestId" yaml:"requestId"`
	RequestUuid     string                 `json:"requestUuid" yaml:"requestId"`
	RequesterId     string                 `json:"requesterId" yaml:"requesterId"`
	RequesterName   string                 `json:"requesterName" yaml:"requesterName"`
	Slack           []SlackResponseSpec    `json:"slack" yaml:"slack"`
	Status          Status                 `json:"status" yaml:"status"`
	StatusUpdatedAt time.Time              `json:"statusUpdatedAt" yaml:"statusUpdatedAt"`
	Telegram        []TelegramResponseSpec `json:"telegram" yaml:"telegram"`
	Type            Platform               `json:"type" yaml:"type"`
}
