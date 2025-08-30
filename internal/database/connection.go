package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/go-sql-driver/mysql"
)

var Connections = map[string]*sql.DB{}
var connectionConfigs = map[string]map[string]ConnectOpts{}

type ConnectOpts struct {
	ConnectionId string

	Host     string
	Port     int
	Username string
	Password string
	Database string
}

func (o *ConnectOpts) Validate() error {
	if o.ConnectionId == "" {
		return fmt.Errorf("failed to receive a connection id")
	}
	if o.Host == "" {
		return fmt.Errorf("failed to receive a host")
	}
	if o.Port < 1024 || o.Port > 65535 {
		return fmt.Errorf("failed to receive a valid port")
	}
	return nil
}

func ConnectMysql(opts ConnectOpts) (*sql.DB, error) {
	addr := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate connection options: %w", err)
	}

	config := mysql.Config{
		User:                 opts.Username,
		Passwd:               opts.Password,
		Net:                  "tcp",
		Addr:                 addr,
		DBName:               opts.Database,
		AllowNativePasswords: true,
		ParseTime:            true,
		MultiStatements:      true,
	}

	connection, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}
	if err := connection.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	Connections[opts.ConnectionId] = connection
	if connectionConfigs["mysql"] == nil {
		connectionConfigs["mysql"] = map[string]ConnectOpts{}
	}
	connectionConfigs["mysql"][opts.ConnectionId] = opts
	return connection, nil
}

func CheckMysqlConnection(connectionId string) error {
	mysqlConnections, ok := connectionConfigs["mysql"]
	if !ok {
		return fmt.Errorf("connection[%s] not found", connectionId)
	}
	if _, ok := mysqlConnections[connectionId]; !ok {
		return fmt.Errorf("connection[%s] not found", connectionId)
	}
	connection, ok := Connections[connectionId]
	if !ok {
		return fmt.Errorf("connection[%s] not found", connectionId)
	}
	if _, err := connection.Exec("SELECT 1"); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			// Check against error code
			if mysqlErr.Number == 4031 {
				return fmt.Errorf("caught inactivity disconnect: %w", err)
			}
		}
		return err
	}
	return nil
}

func RefreshMysqlConnection(connectionId string) error {
	mysqlConnections, ok := connectionConfigs["mysql"]
	if !ok {
		return fmt.Errorf("connection[%s] not found", connectionId)
	}
	if _, ok := mysqlConnections[connectionId]; !ok {
		return fmt.Errorf("connection[%s] not found", connectionId)
	}
	_, ok = Connections[connectionId]
	if !ok {
		return fmt.Errorf("connection[%s] not found", connectionId)
	}
	if connectionConfig, ok := connectionConfigs["mysql"][connectionId]; ok {
		if _, err := ConnectMysql(connectionConfig); err != nil {
			return fmt.Errorf("failed to reconnect: %w", err)
		}
	}
	return nil
}
