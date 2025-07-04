package approver

import "fmt"

func CreateApprovalRequest(req ApprovalRequest) error {
	return Cache.Set(approvalRequestCachePrefix+req.Id, "0", 0)
}

func LoadApprovalRequest(requestId string) (string, error) {
	cacheKey := approvalRequestCachePrefix + requestId
	value, err := Cache.Get(cacheKey)
	if err != nil {
		return "", fmt.Errorf("failed to get req[%s]: %s", cacheKey, err)
	}
	return value, nil
}
