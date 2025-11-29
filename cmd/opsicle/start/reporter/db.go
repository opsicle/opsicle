package reporter

import (
	"fmt"
	"net"
	"opsicle/internal/persistence"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func startMongoReporter(appName string, opts *healthcheckOpts) error {
	logrus.Debugf("connecting to mongodb...")
	mongoInstance := persistence.NewMongo(
		persistence.MongoConnectionOpts{
			AppName:  appName,
			Hosts:    viper.GetStringSlice("mongo-host"),
			IsDirect: true,
		},
		persistence.MongoAuthOpts{
			Password: viper.GetString("mongo-password"),
			Username: viper.GetString("mongo-user"),
		},
		nil,
	)
	if err := mongoInstance.Init(); err != nil {
		return fmt.Errorf("failed to connect to mongo: %w", err)
	}
	logrus.Infof("connected to mongodb")
	opts.status.Set(1)
	go func() {
		for {
			select {
			case <-opts.stopper:
				return
			default:
				<-time.After(opts.interval)
				status := mongoInstance.GetStatus()
				if err := status.GetError(); err != nil {
					opts.status.Set(0)
					logrus.Warnf("mongo[%s] is down, err: '%s' (last updated: %v)", mongoInstance.GetId(), err, status.GetLastUpdatedAt())
					continue
				}
				opts.status.Set(1)
			}
		}
	}()
	return nil
}

func startMysqlReporter(appName string, opts *healthcheckOpts) error {
	logrus.Debugf("connecting to mysql...")
	host := viper.GetString("mysql-host")
	port := viper.GetInt("mysql-port")

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	mysqlInstance := persistence.NewMysql(
		persistence.MysqlConnectionOpts{
			AppName:  appName,
			Host:     addr,
			Database: viper.GetString("mysql-database"),
		},
		persistence.MysqlAuthOpts{
			Password: viper.GetString("mysql-password"),
			Username: viper.GetString("mysql-user"),
		},
		nil,
	)
	if err := mysqlInstance.Init(); err != nil {
		return fmt.Errorf("failed to connect to mysql: %w", err)
	}
	logrus.Infof("connected to mysql")
	opts.status.Set(1)
	go func() {
		for {
			select {
			case <-opts.stopper:
				return
			default:
				<-time.After(opts.interval)
				status := mysqlInstance.GetStatus()
				if err := status.GetError(); err != nil {
					opts.status.Set(0)
					logrus.Warnf("mysql[%s] is down, err: '%s' (last updated: %v)", mysqlInstance.GetId(), err, status.GetLastUpdatedAt())
					continue
				}
				opts.status.Set(1)
			}
		}
	}()
	return nil
}

func startNatsReporter(appName string, opts *healthcheckOpts) error {
	logrus.Debugf("connecting to nats...")
	addr := viper.GetString("nats-addr")
	natsInstance, err := persistence.NewNats(
		persistence.NatsConnectionOpts{
			AppName: appName,
			Host:    addr,
		},
		persistence.NatsAuthOpts{
			NKey: viper.GetString("nats-nkey-value"),
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create nats client: %w", err)
	}
	if err := natsInstance.Init(); err != nil {
		return fmt.Errorf("failed to connect to nats: %w", err)
	}
	logrus.Infof("connected to nats")
	opts.status.Set(1)
	go func() {
		for {
			select {
			case <-opts.stopper:
				return
			default:
				<-time.After(opts.interval)
				status := natsInstance.GetStatus()
				if err := status.GetError(); err != nil {
					opts.status.Set(0)
					logrus.Warnf("nats[%s] is down, err: '%s' (last updated: %v)", natsInstance.GetId(), err, status.GetLastUpdatedAt())
					continue
				}
				opts.status.Set(1)
			}
		}
	}()
	return nil
}

func startRedisReporter(appName string, opts *healthcheckOpts) error {
	logrus.Debugf("connecting to redis...")
	redisInstance := persistence.NewRedis(
		persistence.RedisConnectionOpts{
			AppName: appName,
			Addr:    viper.GetString("redis-addr"),
		},
		persistence.RedisAuthOpts{
			Password: viper.GetString("redis-password"),
			Username: viper.GetString("redis-user"),
		},
		nil,
	)
	if err := redisInstance.Init(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}
	logrus.Infof("connected to redis")
	opts.status.Set(1)
	go func() {
		for {
			select {
			case <-opts.stopper:
				return
			default:
				<-time.After(opts.interval)
				status := redisInstance.GetStatus()
				if err := status.GetError(); err != nil {
					opts.status.Set(0)
					logrus.Warnf("redis[%s] is down, err: '%s' (last updated: %v)", redisInstance.GetId(), err, status.GetLastUpdatedAt())
					continue
				}
				opts.status.Set(1)
			}
		}
	}()
	return nil
}
