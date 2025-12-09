package cache

import (
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
func InitRedis(opts InitRedisOpts) {
	redisOpts := redis.NewOpts{
		Client:      opts.RedisConnection,
		ServiceLogs: opts.ServiceLogs,
	}
	instance = redis.New(redisOpts)
}
