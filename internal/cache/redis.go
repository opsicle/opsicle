package cache

import (
	"fmt"
	"opsicle/internal/common"
	"opsicle/internal/integrations/redis"
	"opsicle/internal/persistence"
)

// InitRedisOpts configures the InitRedis method
type InitRedisOpts struct {
	RedisConnection *persistence.Redis
	ServiceLogs     chan<- common.ServiceLog
}

// InitRedis initialises a singleton instance of a Redis cache
func InitRedis(opts InitRedisOpts) error {
	redisOpts := redis.NewOpts{
		Client:      opts.RedisConnection,
		ServiceLogs: opts.ServiceLogs,
	}
	client, err := redis.New(redisOpts)
	if err != nil {
		return fmt.Errorf("failed to create redis client: %w", err)
	}
	instance = client
	return nil
}
