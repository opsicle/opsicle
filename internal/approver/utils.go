package approver

import (
	"encoding/json"
	"fmt"
	"time"
)

func CreateApproval(approval Approval) error {
	cacheKey := CreateApprovalCacheKey(approval.Spec.Id)
	cacheData, err := json.Marshal(approval)
	if err != nil {
		return fmt.Errorf("failed to marshal approval: %s", err)
	}
	return Cache.Set(cacheKey, string(cacheData), 0)
}

func CreateApprovalRequest(req ApprovalRequest) error {
	cacheKey := CreateApprovalRequestCacheKey(req.Spec.Id, req.Spec.GetUuid())
	cacheData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal approvalRequest: %s", err)
	}
	expiryDuration := time.Duration(req.Spec.TtlSeconds)*time.Second + time.Hour
	return Cache.Set(cacheKey, string(cacheData), expiryDuration)
}

func UpdateApproval(approval Approval) error {
	_, err := LoadApproval(approval)
	if err != nil {
		return err
	}
	return CreateApproval(approval)
}

func UpdateApprovalRequest(req ApprovalRequest) error {
	_, err := LoadApprovalRequest(req)
	if err != nil {
		return err
	}
	return CreateApprovalRequest(req)
}

func LoadApproval(approval Approval) (*ApprovalRequest, error) {
	cacheKey := CreateApprovalRequestCacheKey(approval.Spec.Id)
	value, err := Cache.Get(cacheKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get approval[%s]: %s", cacheKey, err)
	}
	var approvalRequest ApprovalRequest
	if err := json.Unmarshal([]byte(value), &approvalRequest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal approval[%s]: %s", cacheKey, err)
	}
	return &approvalRequest, nil
}

func LoadApprovalRequest(req ApprovalRequest) (*ApprovalRequest, error) {
	cacheKey := CreateApprovalRequestCacheKey(req.Spec.Id, req.Spec.GetUuid())
	value, err := Cache.Get(cacheKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get approvalRequest[%s]: %s", cacheKey, err)
	}
	var approvalRequest ApprovalRequest
	if err := json.Unmarshal([]byte(value), &approvalRequest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal approvalRequest[%s]: %s", cacheKey, err)
	}
	return &approvalRequest, nil
}
