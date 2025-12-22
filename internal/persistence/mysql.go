package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"opsicle/internal/common"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
)

type MysqlConnectionOpts struct {
	AppName             string
	Host                string
	Database            string
	RetryInterval       time.Duration
	HealthcheckInterval time.Duration
}

type MysqlAuthOpts struct {
	Password string
	Username string
}

func NewMysql(
	connectionOpts MysqlConnectionOpts,
	authOpts MysqlAuthOpts,
	serviceLogs *chan common.ServiceLog,
) *Mysql {
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
	output := Mysql{
		healthcheckInterval: healthcheckInterval,
		id:                  getAppName(connectionOpts.AppName),
		options: mysql.Config{
			User:                 authOpts.Username,
			Passwd:               authOpts.Password,
			Net:                  "tcp",
			Addr:                 connectionOpts.Host,
			DBName:               connectionOpts.Database,
			AllowNativePasswords: true,
			ParseTime:            true,
			MultiStatements:      true,
		},
		retryInterval: retryInterval,
		serviceLogs:   serviceLogsInstance,
		status: &Status{
			code:          StatusCodeInitialising,
			lastUpdatedAt: time.Now(),
		},
	}
	output.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "mysql service logs is active for connection[%s]", getAppName(connectionOpts.AppName))
	return &output
}

type Mysql struct {
	id      string
	client  *sql.DB
	options mysql.Config

	healthcheckInterval time.Duration
	retryCount          int
	retryInterval       time.Duration

	serviceLogs chan common.ServiceLog
	status      *Status
}

func (m *Mysql) GetClient() *sql.DB {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelTrace, "mysql[%s] retrieving mysql client...", m.id)
	return m.client
}

func (m *Mysql) GetId() string {
	return m.id
}

func (m *Mysql) GetStatus() *Status {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelTrace, "mysql[%s] retrieving mysql status...", m.id)
	m.status.mutex.Lock()
	defer m.status.mutex.Unlock()
	return &Status{
		code:          m.status.code,
		lastUpdatedAt: m.status.lastUpdatedAt,
		err:           m.status.err,
	}
}

func (m *Mysql) Init() error {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "mysql[%s] is initialising...", m.id)
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
func (m *Mysql) startAutoReconnector() {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "mysql[%s] auto reconnector starting...", m.id)
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
				m.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "failed to reconnect to mysql[%s] after %v attempts: %s", m.id, m.retryCount, err)
				<-time.After(m.retryInterval)
				continue
			}
			if err := m.ping(); err != nil {
				retryLock.Lock()
				m.retryCount++
				retryLock.Unlock()
				m.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "failed to ping mysql[%s] on reconnection after %v attempts: %s", m.id, m.retryCount, err)
				<-time.After(m.retryInterval)
				continue
			}
			retryLock.Lock()
			m.serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "reconnected to mysql after %v attempts and %s", m.retryCount+1, time.Duration(m.retryCount)*m.healthcheckInterval)
			m.retryCount = 0
			retryLock.Unlock()
		}
		<-time.After(m.healthcheckInterval)
	}
}

// startConnectionPinger is designed to be called as a goroutine in the background,
// it does pings to the target database and sets the status to an error state if
// a 'ping' type of request fails
func (m *Mysql) startConnectionPinger() {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "mysql[%s] connection pinger starting...", m.id)
	for {
		if m.status.GetCode() == StatusCodeShuttingDown {
			return
		}
		if err := m.ping(); err != nil {
			m.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "failed to ping mysql[%s]: %s", m.id, err)
		}
		<-time.After(m.healthcheckInterval)
	}
}

func (m *Mysql) Shutdown() error {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "shutting down mysql[%s] connection...", m.id)
	currentStatusCode := m.status.GetCode()
	m.status.set(StatusCodeShuttingDown, nil)
	if currentStatusCode == StatusCodeOk {
		if err := m.client.Close(); err != nil {
			return fmt.Errorf("failed to close mysql connection: %w", err)
		}
		m.client = nil
	}
	return nil
}

func (m *Mysql) connect() error {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "mysql[%s] running Mysql.connect()...", m.id)
	var connectErr error
	if m.client, connectErr = sql.Open("mysql", m.options.FormatDSN()); connectErr != nil {
		m.status.set(StatusCodeConnectError, fmt.Errorf("mysql[%s] failed to connect: %w", m.id, connectErr))
		return m.status.GetError()
	}
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "Mysql.connect() for connection[%s] set status to ok", m.id)
	m.status.set(StatusCodeOk, nil)
	return nil
}

func (m *Mysql) ping() error {
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "mysql[%s] running Mysql.ping()...", m.id)
	isConnectSuccessful := true
	if m.status.GetCode() == StatusCodeConnectError {
		m.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "not pinging mysql[%s] because last error was a connect error", m.id)
		isConnectSuccessful = false
	}
	if !isConnectSuccessful {
		return fmt.Errorf("failed to ping mysql[%s], there is no connection", m.id)
	}
	if _, pingErr := m.client.Exec("SELECT 1"); pingErr != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(pingErr, &mysqlErr) {
			// Check against error code
			if mysqlErr.Number == 4031 {
				m.status.set(StatusCodePingError, fmt.Errorf("mysql[%s] caught inactivity disconnect: %w", m.id, pingErr))
				return m.status.GetError()
			}
		}
		m.status.set(StatusCodePingError, pingErr)
		return m.status.GetError()
	}
	m.serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "Mysql.ping() for connection[%s] set status to ok", m.id)
	m.status.set(StatusCodeOk, nil)
	return nil
}
