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
	client, err := redis.New(redis.NewOpts{
		Addr:           opts.Addr,
		Username:       opts.Username,
		Password:       opts.Password,
		CheckRwEnabled: true,
		ServiceLogs:    &opts.ServiceLogs,
	})
	if err != nil {
		return fmt.Errorf("failed to create redis client: %w", err)
	}
	instance = client
	return nil
}
