package redis

import (
	"fmt"
	"opsicle/internal/common"
	"time"

	"github.com/go-redis/redis/v7"
)

const (
	DefaultNetworkTimeout     = 5 * time.Second
	DefaultNetworkIdleTimeout = 30 * time.Second
)

type Instance struct {
	Client      *redis.Client
	ServiceLogs chan<- common.ServiceLog
}

func (i *Instance) Close() error {
	return i.Client.Close()
}

func (i *Instance) Set(key string, value string, ttl time.Duration) error {
	status := i.Client.Set(key, value, ttl)
	if status.Err() != nil {
		return fmt.Errorf("failed to set key[%s]: %s", key, status.Err())
	}
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "key[%s] creation/update succeeded", key)
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "set key[%s] response: %s", key, status.String())
	return nil
}

func (i *Instance) Get(key string) (string, error) {
	response := i.Client.Get(key)
	if response.Err() != nil {
		return "", fmt.Errorf("failed to get key[%s]: %s", key, response.Err())
	}
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "key[%s] retrieval succeeded", key)
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "get key[%s] response: %s", key, response.String())
	value := response.Val()
	return value, nil
}

func (i *Instance) Scan(pattern string) ([]string, error) {
	response := i.Client.Keys(pattern)
	if response.Err() != nil {
		return nil, fmt.Errorf("failed to list keys[%s]: %s", pattern, response.Err())
	}
	keys := response.Val()
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "keys[%s] scan succeeded", pattern)
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "found %v keys[%s]", len(keys), pattern)
	return keys, nil
}

func (i *Instance) Del(key string) error {
	response := i.Client.Unlink(key)
	if response.Err() != nil {
		return fmt.Errorf("failed to delete key[%s]: %s", key, response.Err())
	}
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "key[%s] deletion succeeded", key)
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "delete key[%s] response: %s", key, response.String())
	return nil
}

func (i *Instance) Ping() error {
	status := i.Client.Ping()
	if status.Err() != nil {
		return fmt.Errorf("failed to ping server: %w", status.Err())
	}
	return nil
}

type NewOpts struct {
	Addr     string
	Username string
	Password string

	CheckRwEnabled bool
	ServiceLogs    *chan<- common.ServiceLog
}

func New(opts NewOpts) (*Instance, error) {
	instance := &Instance{}

	if opts.ServiceLogs == nil {
		serviceLogs := make(chan common.ServiceLog, 8)
		go func() { // noop
			for {
				if _, ok := <-serviceLogs; !ok {
					return
				}
			}
		}()
		instance.ServiceLogs = serviceLogs
	} else {
		instance.ServiceLogs = *opts.ServiceLogs
	}

	redisOptions := &redis.Options{
		Addr:         opts.Addr,
		Username:     opts.Username,
		Password:     opts.Password,
		DB:           0,
		DialTimeout:  DefaultNetworkTimeout,
		ReadTimeout:  DefaultNetworkTimeout,
		WriteTimeout: DefaultNetworkTimeout,
		IdleTimeout:  DefaultNetworkIdleTimeout,
	}
	if opts.ServiceLogs != nil {
		redisOptions.OnConnect = func(c *redis.Conn) error {
			connectionName := c.ClientGetName()
			instance.ServiceLogs <- common.ServiceLogf(
				common.LogLevelDebug,
				"connection[%s] to redis created",
				connectionName.String(),
			)
			return nil
		}
	}
	instance.Client = redis.NewClient(redisOptions)
	if err := instance.Client.Ping().Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis at addr[%s]: %v", opts.Addr, err)
	}
	if opts.CheckRwEnabled {
		now := time.Now().Format("20060102150304")
		testKey := "init-test-" + now
		testValue := "test"
		if status := instance.Client.Set(testKey, testValue, 5*time.Second); status.Err() != nil {
			return nil, fmt.Errorf("failed to set a test key[%s]: %s", testKey, status.Err())
		}
		if res := instance.Client.Get(testKey); res.Err() != nil {
			return nil, fmt.Errorf("failed to receive test key[%s]: %s", testKey, res.Err())
		} else if res.Val() != testValue {
			return nil, fmt.Errorf("failed to receive the correct test value, received '%s'", res.String())
		}
		if res := instance.Client.Unlink(testKey); res.Err() != nil {
			return nil, fmt.Errorf("failed to unlink test key[%s]: %s", testKey, res.Err())
		}
	}

	return instance, nil
}
