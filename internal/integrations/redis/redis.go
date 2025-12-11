package redis

import (
	"errors"
	"fmt"
	"opsicle/internal/common"
	"opsicle/internal/persistence"
	"time"

	"github.com/go-redis/redis/v7"
)

const (
	DefaultNetworkTimeout     = 5 * time.Second
	DefaultNetworkIdleTimeout = 30 * time.Second
)

func IsNilResult(err error) bool {
	return errors.Is(err, redis.Nil)
}

type Instance struct {
	Client      *persistence.Redis
	ServiceLogs chan<- common.ServiceLog
}

func (i *Instance) Close() error {
	return i.Client.GetClient().Close()
}

func (i *Instance) Set(key string, value string, ttl time.Duration) error {
	status := i.Client.GetClient().Set(key, value, ttl)
	if status.Err() != nil {
		return fmt.Errorf("failed to set key[%s]: %w", key, status.Err())
	}
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "key[%s] creation/update succeeded", key)
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "set key[%s] response: %s", key, status.String())
	return nil
}

func (i *Instance) Get(key string) (string, error) {
	response := i.Client.GetClient().Get(key)
	if response.Err() != nil {
		return "", fmt.Errorf("failed to get key[%s]: %w", key, response.Err())
	}
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "key[%s] retrieval succeeded", key)
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "get key[%s] response: %s", key, response.String())
	value := response.Val()
	return value, nil
}

func (i *Instance) Scan(pattern string) ([]string, error) {
	response := i.Client.GetClient().Keys(pattern)
	if response.Err() != nil {
		return nil, fmt.Errorf("failed to list keys[%s]: %w", pattern, response.Err())
	}
	keys := response.Val()
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "keys[%s] scan succeeded", pattern)
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "found %v keys[%s]", len(keys), pattern)
	return keys, nil
}

func (i *Instance) Del(key string) error {
	response := i.Client.GetClient().Unlink(key)
	if response.Err() != nil {
		return fmt.Errorf("failed to delete key[%s]: %w", key, response.Err())
	}
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "key[%s] deletion succeeded", key)
	i.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "delete key[%s] response: %s", key, response.String())
	return nil
}

func (i *Instance) Ping() error {
	status := i.Client.GetClient().Ping()
	if status.Err() != nil {
		return fmt.Errorf("failed to ping server: %w", status.Err())
	}
	return nil
}

type NewOpts struct {
	Client      *persistence.Redis
	ServiceLogs chan<- common.ServiceLog
}

func New(opts NewOpts) *Instance {
	instance := &Instance{
		Client:      opts.Client,
		ServiceLogs: opts.ServiceLogs,
	}
	return instance
}
