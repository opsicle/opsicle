package persistence

import (
	"fmt"
	"opsicle/internal/common"
	"sync"
	"time"

	"github.com/go-redis/redis/v7"
)

const (
	DefaultRedisDialTimeout  = 3 * time.Second
	DefaultRedisReadTimeout  = 3 * time.Second
	DefaultRedisWriteTimeout = 3 * time.Second
	DefaultRedisIdleTimeout  = 3 * time.Second
)

type RedisConnectionOpts struct {
	AppName             string
	Addr                string
	DB                  int
	RetryInterval       time.Duration
	HealthcheckInterval time.Duration
}

type RedisAuthOpts struct {
	Username string
	Password string
}

func NewRedis(
	connectionOpts RedisConnectionOpts,
	authOpts RedisAuthOpts,
	serviceLogs *chan common.ServiceLog,
) *Redis {
	serviceLogsInstance := common.GetNoopServiceLog()
	if serviceLogs != nil {
		serviceLogsInstance = *serviceLogs
	}
	healthcheckInterval := DefaultHealthcheckInterval
	if connectionOpts.HealthcheckInterval != 0 {
		healthcheckInterval = connectionOpts.HealthcheckInterval
	}
	retryInterval := DefaultRetryInterval
	if connectionOpts.RetryInterval != 0 {
		retryInterval = connectionOpts.RetryInterval
	}

	redisOptions := &redis.Options{
		Addr:         connectionOpts.Addr,
		DB:           connectionOpts.DB,
		Username:     authOpts.Username,
		Password:     authOpts.Password,
		DialTimeout:  DefaultRedisDialTimeout,
		ReadTimeout:  DefaultRedisReadTimeout,
		WriteTimeout: DefaultRedisWriteTimeout,
		IdleTimeout:  DefaultRedisIdleTimeout,
		OnConnect: func(c *redis.Conn) error {
			connectionName := c.ClientGetName()
			serviceLogsInstance <- common.ServiceLogf(
				common.LogLevelDebug,
				"connection[%s] to redis created",
				connectionName.String(),
			)
			return nil
		},
	}
	output := Redis{
		client:  redis.NewClient(redisOptions),
		id:      getAppName(connectionOpts.AppName),
		options: redisOptions,

		healthcheckInterval: healthcheckInterval,
		retryInterval:       retryInterval,

		serviceLogs: serviceLogsInstance,
		status: &Status{
			code:          StatusCodeInitialising,
			lastUpdatedAt: time.Now(),
		},
	}
	return &output
}

type Redis struct {
	client  *redis.Client
	id      string
	options *redis.Options

	healthcheckInterval time.Duration
	retryCount          int
	retryInterval       time.Duration

	serviceLogs chan<- common.ServiceLog
	status      *Status
}

func (r *Redis) GetClient() *redis.Client {
	r.serviceLogs <- common.ServiceLogf(common.LogLevelTrace, "redis[%s] retrieving redis client...", r.id)
	return r.client
}

func (r *Redis) GetId() string {
	return r.id
}

func (r *Redis) GetStatus() *Status {
	r.serviceLogs <- common.ServiceLogf(common.LogLevelTrace, "redis[%s] retrieving nats status...", r.id)
	r.status.mutex.Lock()
	defer r.status.mutex.Unlock()
	return &Status{
		code:          r.status.code,
		lastUpdatedAt: r.status.lastUpdatedAt,
		err:           r.status.err,
	}
}

func (r *Redis) Init() error {
	r.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "redis[%s] is initialising...", r.id)
	if err := r.connect(); err != nil {
		return r.status.err
	}
	if err := r.ping(); err != nil {
		return err
	}
	go r.startAutoReconnector()
	go r.startConnectionPinger()
	return nil
}

func (r *Redis) Shutdown() error {
	r.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "shutting down redis connection...")
	currentStatusCode := r.status.GetCode()
	r.status.set(StatusCodeShuttingDown, nil)
	if currentStatusCode == StatusCodeOk {
		if err := r.client.Close(); err != nil {
			return fmt.Errorf("failed to disconnect redis: %w", err)
		}
		r.client = nil
	}
	return nil
}

// startAutoReconnector is designed to be called as a goroutine in the background,
// it checks for an errored status and attempts to reconnect to the database until
// it's successful again
func (r *Redis) startAutoReconnector() {
	r.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "redis[%s] auto reconnector starting...", r.id)
	var retryLock sync.Mutex

	for {
		if r.status.GetCode() == StatusCodeShuttingDown {
			return
		}
		if r.status.GetError() != nil {
			if err := r.connect(); err != nil {
				retryLock.Lock()
				r.retryCount++
				retryLock.Unlock()
				r.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "failed to reconnect to redis[%s] after %v attempts: %s", r.id, r.retryCount, err)
				<-time.After(r.retryInterval)
				continue
			}
			if err := r.ping(); err != nil {
				retryLock.Lock()
				r.retryCount++
				retryLock.Unlock()
				r.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "failed to ping redis[%s] on reconnection after %v attempts: %s", r.id, r.retryCount, err)
				<-time.After(r.retryInterval)
				continue
			}
			retryLock.Lock()
			r.retryCount = 0
			retryLock.Unlock()
		}
		<-time.After(r.healthcheckInterval)
	}
}

// startConnectionPinger is designed to be called as a goroutine in the background,
// it does pings to the target database and sets the status to an error state if
// a 'ping' type of request fails
func (r *Redis) startConnectionPinger() {
	r.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "redis[%s] connection pinger starting...", r.id)
	for {
		if r.status.GetCode() == StatusCodeShuttingDown {
			return
		}
		if err := r.ping(); err != nil {
			r.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "failed to ping redis[%s]: %s", r.id, err)
		}
		<-time.After(r.healthcheckInterval)
	}
}

func (r *Redis) connect() error {
	r.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "redis[%s] running Redis.connect()...", r.id)
	now := time.Now().Format("20060102150304")
	testKey := "connect-test-" + now
	testValue := "test"
	if status := r.client.Set(testKey, testValue, 5*time.Second); status.Err() != nil {
		r.status.set(StatusCodeConnectError, fmt.Errorf("redis[%s] failed to SET: %w", r.id, status.Err()))
		return r.status.GetError()
	}
	if res := r.client.Get(testKey); res.Err() != nil {
		r.status.set(StatusCodeConnectError, fmt.Errorf("redis[%s] failed to GET: %w", r.id, res.Err()))
		return r.status.GetError()
	} else if res.Val() != testValue {
		r.status.set(StatusCodeConnectError, fmt.Errorf("redis[%s] failed to reconcile SET/GET value", r.id))
		return r.status.GetError()
	}
	if res := r.client.Unlink(testKey); res.Err() != nil {
		r.status.set(StatusCodeConnectError, fmt.Errorf("redis[%s] failed to DEL: %w", r.id, res.Err()))
		return r.status.GetError()
	}
	r.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "Redis.connect() for connection[%s] set status to ok", r.id)
	r.status.set(StatusCodeOk, nil)
	return nil
}

func (r *Redis) ping() error {
	r.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "redis[%s] running Redis.ping()...", r.id)

	isConnectSuccessful := true
	if r.status.GetCode() == StatusCodeConnectError {
		r.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "not pinging redis[%s] because last error was a connect error", r.id)
		isConnectSuccessful = false
	}
	if !isConnectSuccessful {
		return fmt.Errorf("failed to ping redis[%s], there is no connection", r.id)
	}

	if err := r.client.Ping().Err(); err != nil {
		r.status.set(StatusCodePingError, fmt.Errorf("redis[%s] connection closed, last error: %w", r.id, err))
		return r.status.GetError()
	}
	r.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "Redis.ping() for connection[%s] set status to ok", r.id)
	r.status.set(StatusCodeOk, nil)
	return nil
}
