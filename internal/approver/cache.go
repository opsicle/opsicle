package approver

import (
	"strings"
)

type pendingMfa struct {
	ApprovalRequestMessageId string `json:"approvalRequestMessageId"`
	ChatId                   string `json:"chatId"`
	MfaSeed                  string `json:"mfaSeed"`
	RequestId                string `json:"requestId"`
	RequestUuid              string `json:"requestUuid"`
	UserId                   string `json:"userId"`
}

func CreateApprovalCacheKey(requestIdentifiers ...string) string {
	cacheKeys := []string{approvalCachePrefix}
	cacheKeys = append(cacheKeys, requestIdentifiers...)
	return strings.Join(cacheKeys, ":")
}

func CreateApprovalRequestCacheKey(requestIdentifiers ...string) string {
	cacheKeys := []string{approvalRequestCachePrefix}
	cacheKeys = append(cacheKeys, requestIdentifiers...)
	return strings.Join(cacheKeys, ":")
}

func CreatePendingMfaCacheKey(requestIdentifiers ...string) string {
	cacheKeys := []string{pendingMfaCachePrefix}
	cacheKeys = append(cacheKeys, requestIdentifiers...)
	return strings.Join(cacheKeys, ":")
}

func StripCacheKeyPrefix(cacheKey string) string {
	requestIdentifier := strings.Split(cacheKey, ":")
	if len(requestIdentifier) == 2 {
		return requestIdentifier[1]
	}
	return strings.Join(requestIdentifier[1:], ":")
}
