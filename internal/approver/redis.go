package approver

import (
	"fmt"
	"opsicle/internal/common"
	"opsicle/internal/integrations/redis"
)

// InitRedisCacheOpts configures the InitRedisCache method
type InitRedisCacheOpts struct {
	Addr     string
	Username string
	Password string

	ServiceLogs chan<- common.ServiceLog
}

// InitRedisCache initialises a singleton instance of a Redis cache
func InitRedisCache(opts InitRedisCacheOpts) error {
	client, err := redis.New(redis.NewOpts{
		Addr:           opts.Addr,
		Username:       opts.Username,
		Password:       opts.Password,
		CheckRwEnabled: true,
		ServiceLogs:    &opts.ServiceLogs,
	})
	if err != nil {
		return fmt.Errorf("failed to create redis client: %s", err)
	}
	Cache = client
	return nil
}
