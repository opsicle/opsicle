package persistence

import (
	"fmt"
	"opsicle/internal/common"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

type NatsConnectionOpts struct {
	AppName             string
	Host                string
	RetryInterval       time.Duration
	HealthcheckInterval time.Duration
}

type NatsAuthOpts struct {
	NKey     string
	Username string
	Password string
}

func NewNats(
	connectionOpts NatsConnectionOpts,
	authOpts NatsAuthOpts,
	serviceLogs *chan common.ServiceLog,
) (*Nats, error) {
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
	output := Nats{
		addr:                connectionOpts.Host,
		healthcheckInterval: healthcheckInterval,
		id:                  getAppName(connectionOpts.AppName),
		options:             []nats.Option{},
		retryInterval:       retryInterval,
		serviceLogs:         serviceLogsInstance,
		status: &Status{
			code:          StatusCodeInitialising,
			lastUpdatedAt: time.Now(),
		},
	}
	if authOpts.NKey != "" {
		keyPair, err := nkeys.FromSeed([]byte(authOpts.NKey))
		if err != nil {
			return nil, fmt.Errorf("failed to generate keypair from nkey: %w", err)
		}
		publicKey, err := keyPair.PublicKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate public key from nkey: %w", err)
		}
		output.options = append(output.options, nats.Nkey(publicKey, keyPair.Sign))
	} else if authOpts.Username != "" && authOpts.Password != "" {
		output.options = append(output.options, nats.UserInfo(authOpts.Username, authOpts.Password))
	} else {
		return nil, fmt.Errorf("failed to receive an auth method")
	}
	output.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "nats service logs is active for connection[%s]", getAppName(connectionOpts.AppName))
	return &output, nil
}

type Nats struct {
	id      string
	client  *nats.Conn
	addr    string
	options []nats.Option

	healthcheckInterval time.Duration
	retryCount          int
	retryInterval       time.Duration

	serviceLogs chan common.ServiceLog
	status      *Status
}

func (n *Nats) GetClient() *nats.Conn {
	n.serviceLogs <- common.ServiceLogf(common.LogLevelTrace, "nats[%s] retrieving nats client...", n.id)
	return n.client
}

func (n *Nats) GetId() string {
	return n.id
}

func (n *Nats) GetStreamingClient() (nats.JetStreamContext, error) {
	n.serviceLogs <- common.ServiceLogf(common.LogLevelTrace, "nats[%s] retrieving nats jetstream client...", n.id)
	js, err := n.client.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to get nats[%s] jetstream context: %w", n.id, err)
	}
	return js, nil
}

func (n *Nats) GetStatus() *Status {
	n.serviceLogs <- common.ServiceLogf(common.LogLevelTrace, "nats[%s] retrieving nats status...", n.id)
	n.status.mutex.Lock()
	defer n.status.mutex.Unlock()
	return &Status{
		code:          n.status.code,
		lastUpdatedAt: n.status.lastUpdatedAt,
		err:           n.status.err,
	}
}

func (n *Nats) Init() error {
	n.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "nats[%s] is initialising...", n.id)
	if err := n.connect(); err != nil {
		return n.status.err
	}
	if err := n.ping(); err != nil {
		return err
	}
	go n.startAutoReconnector()
	go n.startConnectionPinger()
	return nil
}

func (n *Nats) Shutdown() error {
	n.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "shutting down nats connection...")
	currentStatusCode := n.status.GetCode()
	n.status.set(StatusCodeShuttingDown, nil)
	if currentStatusCode == StatusCodeOk {
		if err := n.client.Flush(); err != nil {
			n.serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to flush nats")
		}
		n.client.Close()
		n.client = nil
	}
	return nil
}

// startAutoReconnector is designed to be called as a goroutine in the background,
// it checks for an errored status and attempts to reconnect to the database until
// it's successful again
func (n *Nats) startAutoReconnector() {
	n.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "nats[%s] auto reconnector starting...", n.id)
	var retryLock sync.Mutex

	for {
		if n.status.GetCode() == StatusCodeShuttingDown {
			return
		}
		if n.status.GetError() != nil {
			if err := n.connect(); err != nil {
				retryLock.Lock()
				n.retryCount++
				retryLock.Unlock()
				n.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "failed to reconnect to nats[%s] after %v attempts: %s", n.id, n.retryCount, err)
				<-time.After(n.retryInterval)
				continue
			}
			if err := n.ping(); err != nil {
				retryLock.Lock()
				n.retryCount++
				retryLock.Unlock()
				n.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "failed to ping nats[%s] on reconnection after %v attempts: %s", n.id, n.retryCount, err)
				<-time.After(n.retryInterval)
				continue
			}
			retryLock.Lock()
			n.serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "reconnected to nats after %v attempts and %s", n.retryCount+1, time.Duration(n.retryCount)*n.healthcheckInterval)
			n.retryCount = 0
			retryLock.Unlock()
		}
		<-time.After(n.healthcheckInterval)
	}
}

// startConnectionPinger is designed to be called as a goroutine in the background,
// it does pings to the target database and sets the status to an error state if
// a 'ping' type of request fails
func (n *Nats) startConnectionPinger() {
	n.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "nats[%s] connection pinger starting...", n.id)
	for {
		if n.status.GetCode() == StatusCodeShuttingDown {
			return
		}
		if err := n.ping(); err != nil {
			n.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "failed to ping nats[%s]: %s", n.id, err)
		}
		<-time.After(n.healthcheckInterval)
	}
}

func (n *Nats) connect() error {
	n.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "nats[%s] running Nats.connect()...", n.id)
	var connectErr error
	if n.client, connectErr = nats.Connect("nats://"+n.addr, n.options...); connectErr != nil {
		n.status.set(StatusCodeConnectError, fmt.Errorf("nats[%s] failed to connect: %w", n.id, connectErr))
		return n.status.GetError()
	}
	if !n.client.IsConnected() {
		n.status.set(StatusCodeConnectError, fmt.Errorf("nats[%s] failed to verify connection", n.id))
		return n.status.GetError()
	}
	n.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "Nats.connect() for connection[%s] set status to ok", n.id)
	n.status.set(StatusCodeOk, nil)
	return nil
}

func (n *Nats) ping() error {
	n.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "nats[%s] running Nats.ping()...", n.id)
	isConnectSuccessful := true
	if n.status.GetCode() == StatusCodeConnectError {
		n.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "not pinging nats[%s] because last error was a connect error", n.id)
		isConnectSuccessful = false
	}
	if !isConnectSuccessful {
		return fmt.Errorf("failed to ping nats[%s], there is no connection", n.id)
	}
	if n.client.IsClosed() {
		n.status.set(StatusCodePingError, fmt.Errorf("nats[%s] connection closed, last error: %w", n.id, n.client.LastError()))
		return n.status.GetError()
	}
	if n.client.IsDraining() {
		n.status.set(StatusCodePingError, fmt.Errorf("nats[%s] connection is being drained, last error: %w", n.id, n.client.LastError()))
		return n.status.GetError()
	}
	if n.client.IsReconnecting() {
		n.status.set(StatusCodePingError, fmt.Errorf("nats[%s] connection is re-establishing, last error: %w", n.id, n.client.LastError()))
		return n.status.GetError()
	}
	n.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "Nats.ping() for connection[%s] set status to ok", n.id)
	n.status.set(StatusCodeOk, nil)
	return nil
}
