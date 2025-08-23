package database

import (
	"database/sql"
	"fmt"
	"net"
	"strconv"

	"github.com/go-sql-driver/mysql"
)

var Connections = map[string]*sql.DB{}

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
	return connection, nil
}
