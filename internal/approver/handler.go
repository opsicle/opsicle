package approver

import (
	"encoding/json"
	"fmt"
	"strings"
)

func CreateApprovalRequest(req ApprovalRequest) error {
	cacheKey := CreateCacheKey(req.Spec.Id, req.Spec.GetUuid())
	cacheData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal approval request: %s", err)
	}
	return Cache.Set(cacheKey, string(cacheData), 0)
}

func UpdateApprovalRequest(req ApprovalRequest) error {
	_, err := LoadApprovalRequest(req)
	if err != nil {
		return err
	}
	return CreateApprovalRequest(req)
}

func LoadApprovalRequest(req ApprovalRequest) (*ApprovalRequest, error) {
	cacheKey := CreateCacheKey(req.Spec.Id, req.Spec.GetUuid())
	value, err := Cache.Get(cacheKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get req[%s]: %s", cacheKey, err)
	}
	fmt.Println(value)
	var approvalRequest ApprovalRequest
	if err := json.Unmarshal([]byte(value), &approvalRequest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal req[%s]: %s", cacheKey, err)
	}
	return &approvalRequest, nil
}

func CreateCacheKey(requestIdentifiers ...string) string {
	cacheKeys := []string{approvalRequestCachePrefix}
	cacheKeys = append(cacheKeys, requestIdentifiers...)
	return strings.Join(cacheKeys, ":")
}
