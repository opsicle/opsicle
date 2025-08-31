package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Connections = map[string]*sql.DB{}
var ConnectionsMongo = map[string]*mongo.Client{}
var connectionConfigs = map[string]map[string]ConnectOpts{}

type ConnectOpts struct {
	// ConnectionId is used by this package internally to track a
	// connection and it's health
	ConnectionId string

	// Host specifies the hostname where the database is reachable at
	Host string

	// Hosts is applicable only in systems that support mutliple
	// brokers/nodes (eg. Mongo). Unlike the `Host` variable, entries
	// in this slice should contain the port as well
	Hosts []string

	// Port specifies the port which the database is listening on.
	// Ignored if the `Hosts` variable is specified
	Port int

	// Username is the username to use for authenticating with the databse
	Username string

	// Password is the password to use for authenticating with the databse
	Password string

	// Database is the name of the database to use upon connection
	Database string

	// Opts contains additional connection options that are used in different
	// ways by the different databases. Check with the initialiser's `Connect*`
	// function documentation to see how this variable is used
	Opts map[string]any
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

// ConnectMongo creates a connection to a MongoDB instance.
//
// The `opts.Opts` supports the following parameters:
// 1. "direct" : bool
func ConnectMongo(opts ConnectOpts) (*mongo.Client, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate connection options: %w", err)
	}
	mongoCredentials := options.Credential{
		Username: opts.Username,
		Password: opts.Password,
	}
	mongoHosts := []string{}
	if opts.Hosts != nil {
		mongoHosts = append(mongoHosts, opts.Hosts...)
	} else {
		if opts.Host == "" {
			return nil, fmt.Errorf("host is unspecified")
		} else if opts.Port == 0 {
			return nil, fmt.Errorf("port is unspecified")
		}
		addr := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
		mongoHosts = append(mongoHosts, addr)
	}
	mongoOpts := options.Client().
		SetHosts(mongoHosts).
		SetAuth(mongoCredentials)
	if opts.Opts != nil {
		if val, ok := opts.Opts["direct"]; ok {
			if value, ok := val.(bool); ok {
				mongoOpts = mongoOpts.SetDirect(value)
			} else {
				return nil, fmt.Errorf("expected 'direct' to be of type <bool>")
			}
		}
	}
	mongoContext, cancelMongoContext := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelMongoContext()
	connection, err := mongo.Connect(mongoContext, mongoOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection to mongo: %w", err)
	}
	if err := connection.Ping(mongoContext, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongo: %w", err)
	}
	ConnectionsMongo[opts.ConnectionId] = connection
	if _, ok := connectionConfigs["mongo"]; !ok {
		connectionConfigs["mongo"] = map[string]ConnectOpts{}
	}
	connectionConfigs["mongo"][opts.ConnectionId] = opts
	return connection, nil
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
