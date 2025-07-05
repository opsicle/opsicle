package approver

import (
	"time"
)

var Cache cache

type cache interface {
	Set(key string, value string, ttl time.Duration) (err error)
	Get(key string) (value string, err error)
	Scan(prefix string) (keys []string, err error)
	Del(key string) (err error)
}
