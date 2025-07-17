package approver

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/approvals"
	"time"
)

type ApprovalRequest struct {
	Spec approvals.RequestSpec `json:"spec" yaml:"spec"`
}

func (req *ApprovalRequest) Create() error {
	cacheKey := CreateApprovalRequestCacheKey(req.Spec.GetUuid())
	cacheData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal approvalRequest[%s]: %s", req.Spec.GetUuid(), err)
	}
	expiryDuration := time.Duration(req.Spec.TtlSeconds)*time.Second + time.Hour
	if err := Cache.Set(cacheKey, string(cacheData), expiryDuration); err != nil {
		return fmt.Errorf("failed to set cache for approvalRequest[%s]: %s", req.Spec.GetUuid(), err)
	}
	return nil
}

func (req *ApprovalRequest) Exists() bool {
	cacheKey := CreateApprovalRequestCacheKey(req.Spec.GetUuid())
	if _, err := Cache.Get(cacheKey); err != nil {
		return false
	}
	return true
}

func (req *ApprovalRequest) GetRedacted() ApprovalRequest {
	approvalRequest := *req
	redactedText := "<REDACTED>"
	for i, target := range approvalRequest.Spec.Slack {
		for j, authorizedResponder := range target.AuthorizedResponders {
			if authorizedResponder.MfaSeed != nil {
				approvalRequest.Spec.Slack[i].AuthorizedResponders[j].MfaSeed = &redactedText
			}
		}
	}
	for i, target := range approvalRequest.Spec.Telegram {
		for j, authorizedResponder := range target.AuthorizedResponders {
			if authorizedResponder.MfaSeed != nil {
				approvalRequest.Spec.Telegram[i].AuthorizedResponders[j].MfaSeed = &redactedText
			}
		}
	}
	return approvalRequest
}

func (req *ApprovalRequest) Update() error {
	if isExists := req.Exists(); !isExists {
		return fmt.Errorf("failed to find an existing approvalRequest[%s]", req.Spec.GetUuid())
	}
	if err := req.Create(); err != nil {
		return fmt.Errorf("failed to create approvalRequest[%s]: %s", req.Spec.GetUuid(), err)
	}
	return nil
}

func (req *ApprovalRequest) Load() error {
	cacheKey := CreateApprovalRequestCacheKey(req.Spec.GetUuid())
	value, err := Cache.Get(cacheKey)
	if err != nil {
		return fmt.Errorf("failed to get approvalRequest[%s]: %s", cacheKey, err)
	}
	if err := json.Unmarshal([]byte(value), req); err != nil {
		return fmt.Errorf("failed to unmarshal approvalRequest[%s]: %s (full object: %s)", cacheKey, err, value)
	}
	return nil
}
