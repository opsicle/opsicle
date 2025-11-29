package persistence

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

func CheckMongoConnection(connectionId string) error {
	mongoConnections, ok := connectionConfigs["mongo"]
	if !ok {
		return fmt.Errorf("connection[%s] not found", connectionId)
	}
	if _, ok := mongoConnections[connectionId]; !ok {
		return fmt.Errorf("connection[%s] not found", connectionId)
	}
	connection, ok := ConnectionsMongo[connectionId]
	if !ok {
		return fmt.Errorf("connection[%s] not found", connectionId)
	}
	pingCtx, cancelPing := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelPing()
	if err := connection.Ping(pingCtx, nil); err != nil {
		return fmt.Errorf("connection[%s] disconnected: %w", connectionId, err)
	}
	return nil
}

func RefreshMongoConnection(connectionId string) error {
	mongoConnections, ok := connectionConfigs["mongo"]
	if !ok {
		return fmt.Errorf("connection[%s] not found", connectionId)
	}
	if _, ok := mongoConnections[connectionId]; !ok {
		return fmt.Errorf("connection[%s] not found", connectionId)
	}
	_, ok = ConnectionsMongo[connectionId]
	if !ok {
		return fmt.Errorf("connection[%s] not found", connectionId)
	}
	if connectionConfig, ok := connectionConfigs["mongo"][connectionId]; ok {
		if _, err := ConnectMongo(connectionConfig); err != nil {
			return fmt.Errorf("failed to reconnect: %w", err)
		}
	}
	return nil
}
