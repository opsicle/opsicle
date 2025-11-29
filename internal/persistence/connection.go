package persistence

import (
	"database/sql"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

var ConnectionsMysql = map[string]*sql.DB{}
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
