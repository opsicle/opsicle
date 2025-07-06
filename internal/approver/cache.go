package approver

import (
	"strings"
	"time"
)

var Cache cache

type cache interface {
	Set(key string, value string, ttl time.Duration) (err error)
	Get(key string) (value string, err error)
	Scan(prefix string) (keys []string, err error)
	Del(key string) (err error)
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
