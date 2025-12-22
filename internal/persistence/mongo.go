package persistence

import (
	"context"
	"fmt"
	"opsicle/internal/common"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoConnectionOpts struct {
	AppName             string
	Hosts               []string
	Database            string
	IsDirect            bool
	RetryInterval       time.Duration
	HealthcheckInterval time.Duration
}

type MongoAuthOpts struct {
	AuthMechanism string
	AuthSource    string
	Password      string
	Username      string
}

func (mao MongoAuthOpts) toNative() options.Credential {
	return options.Credential{
		AuthMechanism: mao.AuthMechanism,
		AuthSource:    mao.AuthSource,
		Password:      mao.Password,
		Username:      mao.Username,
	}
}

func NewMongo(
	connectionOpts MongoConnectionOpts,
	authOpts MongoAuthOpts,
	serviceLogs *chan common.ServiceLog,
) *Mongo {
	serviceLogsInstance := common.GetNoopServiceLog()
	if serviceLogs != nil {
		serviceLogsInstance = *serviceLogs
	}
	healthcheckInterval := 3 * time.Second
	if connectionOpts.HealthcheckInterval != 0 {
		healthcheckInterval = connectionOpts.HealthcheckInterval
	}
	retryInterval := 3 * time.Second
	if connectionOpts.RetryInterval != 0 {
		retryInterval = connectionOpts.RetryInterval
	}
	output := Mongo{
		healthcheckInterval: healthcheckInterval,
		id:                  getAppName(connectionOpts.AppName),
		options: options.Client().
			SetHosts(connectionOpts.Hosts).
			SetDirect(connectionOpts.IsDirect).
			SetAuth(authOpts.toNative()).
			SetAppName(connectionOpts.AppName),
		retryInterval: retryInterval,
		serviceLogs:   serviceLogsInstance,
		status: &Status{
			code:          StatusCodeInitialising,
			lastUpdatedAt: time.Now(),
		},
	}
	output.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "mongo service logs is active for connection[%s]", getAppName(connectionOpts.AppName))
	return &output
}

type Mongo struct {
	id      string
	client  *mongo.Client
	options *options.ClientOptions

	healthcheckInterval time.Duration
	retryCount          int
	retryInterval       time.Duration

	serviceLogs chan common.ServiceLog
	status      *Status
}

func (m *Mongo) GetClient() *mongo.Client {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelTrace, "retrieving mongo client...")
	return m.client
}

func (m *Mongo) GetId() string {
	return m.id
}

func (m *Mongo) GetStatus() *Status {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelTrace, "retrieving mongo status...")
	m.status.mutex.Lock()
	defer m.status.mutex.Unlock()
	return &Status{
		code:          m.status.code,
		lastUpdatedAt: m.status.lastUpdatedAt,
		err:           m.status.err,
	}
}

func (m *Mongo) Init() error {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "mongo is initialising...")
	if err := m.connect(); err != nil {
		return m.status.err
	}
	if err := m.ping(); err != nil {
		return err
	}
	go m.startAutoReconnector()
	go m.startConnectionPinger()
	return nil
}

// startAutoReconnector is designed to be called as a goroutine in the background,
// it checks for an errored status and attempts to reconnect to the database until
// it's successful again
func (m *Mongo) startAutoReconnector() {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "starting mongo auto-reconnector...")
	var retryLock sync.Mutex

	for {
		if m.status.GetCode() == StatusCodeShuttingDown {
			return
		}
		if m.status.GetError() != nil {
			if err := m.connect(); err != nil {
				retryLock.Lock()
				m.retryCount++
				retryLock.Unlock()
				m.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "failed to reconnect to mongo after %v attempts: %s", m.retryCount, err)
				<-time.After(m.retryInterval)
				continue
			}
			if err := m.ping(); err != nil {
				retryLock.Lock()
				m.retryCount++
				retryLock.Unlock()
				m.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "failed to ping mongo on reconnection after %v attempts: %s", m.retryCount, err)
				<-time.After(m.retryInterval)
				continue
			}
			retryLock.Lock()
			m.serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "reconnected to mongo after %v attempts and %s", m.retryCount+1, time.Duration(m.retryCount)*m.healthcheckInterval)
			m.retryCount = 0
			retryLock.Unlock()
		}
		<-time.After(m.healthcheckInterval)
	}
}

// startConnectionPinger is designed to be called as a goroutine in the background,
// it does pings to the target database and sets the status to an error state if
// a 'ping' type of request fails
func (m *Mongo) startConnectionPinger() {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "starting mongo connection pinger...")
	for {
		if m.status.GetCode() == StatusCodeShuttingDown {
			return
		}
		if err := m.ping(); err != nil {
			m.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "failed to ping mongo: %s", err)
		}
		<-time.After(m.healthcheckInterval)
	}
}

func (m *Mongo) Shutdown() error {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "shutting down mongo connection...")
	currentStatusCode := m.status.GetCode()
	m.status.set(StatusCodeShuttingDown, nil)
	if currentStatusCode == StatusCodeOk {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := m.client.Disconnect(ctx); err != nil {
			return fmt.Errorf("failed to disconnect mongo: %w", err)
		}
		m.client = nil
	}
	return nil
}

func (m *Mongo) connect() (err error) {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "running Mongo.connect()...")
	connectCtx, cancelConnect := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelConnect()

	var connectErr error
	m.options.SetConnectTimeout(3 * time.Second)
	// there's weird behaviour here where .Connect doesn't actually do any I/O so it's
	// impossible to verify that the provided connection/credential parameters are up/valid
	if m.client, connectErr = mongo.Connect(connectCtx, m.options); connectErr != nil {
		m.status.set(StatusCodeConnectError, fmt.Errorf("failed to create mongo client: %w", connectErr))
		err = m.status.GetError()
	}
	// therefore we need to re-implement a ping here
	pingCtx, cancelPing := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelPing()
	if pingErr := m.client.Ping(pingCtx, nil); pingErr != nil {
		m.client = nil
		m.status.set(StatusCodeConnectError, pingErr)
		return m.status.GetError()
	}
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "Mongo.connect() set status to ok")
	m.status.set(StatusCodeOk, nil)
	return err
}

func (m *Mongo) ping() error {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "running Mongo.ping()...")
	isConnectSuccessful := true
	if m.status.GetCode() == StatusCodeConnectError {
		m.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "not pinging because last error was a connect error")
		isConnectSuccessful = false
	}
	if !isConnectSuccessful {
		return fmt.Errorf("failed to ping mongo, there is no connection")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if pingErr := m.client.Ping(ctx, nil); pingErr != nil {
		m.status.set(StatusCodePingError, pingErr)
		return m.status.GetError()
	}
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "Mongo.ping() set status to ok")
	m.status.set(StatusCodeOk, nil)
	return nil
}
