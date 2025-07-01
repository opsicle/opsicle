package approver

import (
	"fmt"
	"time"

	"github.com/go-redis/redis/v7"
)

var RedisCache *redisCache

type redisCache struct {
	Client *redis.Client
}

type InitRedisCacheOpts struct {
	Addr     string
	Username string
	Password string
}

func InitRedisCache(opts InitRedisCacheOpts) error {
	client := redis.NewClient(&redis.Options{
		Addr:     opts.Addr,
		Username: opts.Username,
		Password: opts.Password,
		DB:       0,
	})
	if err := client.Ping().Err(); err != nil {
		return fmt.Errorf("failed to connect to redis at addr[%s]: %v", opts.Addr, err)
	}
	now := time.Now().Format("20060102150304")
	testKey := "init-test-" + now
	testValue := "test"
	if status := client.Set(testKey, testValue, 5*time.Second); status.Err() != nil {
		return fmt.Errorf("failed to set a test key[%s]: %s", testKey, status.Err())
	}
	if res := client.Get(testKey); res.Err() != nil {
		return fmt.Errorf("failed to receive test key[%s]: %s", testKey, res.Err())
	} else if res.Val() != testValue {
		return fmt.Errorf("failed to receive the correct test value, received '%s'", res.String())
	}
	if res := client.Unlink(testKey); res.Err() != nil {
		return fmt.Errorf("failed to unlink test key[%s]: %s", testKey, res.Err())
	}
	RedisCache = &redisCache{
		Client: client,
	}
	return nil
}
