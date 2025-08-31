package cache

import (
	"fmt"
	"opsicle/internal/common"
	"opsicle/internal/integrations/redis"
)

// InitRedisOpts configures the InitRedis method
type InitRedisOpts struct {
	Addr     string
	Username string
	Password string

	ServiceLogs chan<- common.ServiceLog
}

// InitRedis initialises a singleton instance of a Redis cache
func InitRedis(opts InitRedisOpts) error {
	redisOpts := redis.NewOpts{
		Addr:           opts.Addr,
		Username:       opts.Username,
		Password:       opts.Password,
		CheckRwEnabled: true,
	}
	if opts.ServiceLogs != nil {
		redisOpts.ServiceLogs = &opts.ServiceLogs
	} else {
		initNoopServiceLog()
		var serviceLogs chan<- common.ServiceLog = noopServiceLog
		redisOpts.ServiceLogs = &serviceLogs
		go startNoopServiceLog()
		defer stopNoopServiceLog()
	}
	client, err := redis.New(redisOpts)
	if err != nil {
		return fmt.Errorf("failed to create redis client: %w", err)
	}
	instance = client
	return nil
}
