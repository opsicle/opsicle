package audit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Log(log LogEntry) error {
	if client != nil {
		return client.Log(log)
	}
	logrus.Warnf("audit.Log called without a valid logger")
	return ErrorNotInitialized
}

func GetByEntity(entityId string, entityType EntityType, cursor time.Time, limit int64) (LogEntries, error) {
	if client != nil {
		return client.GetByEntity(entityId, entityType, cursor, limit)
	}
	logrus.Warnf("audit.GetByEntity called without a valid logger")
	return nil, ErrorNotInitialized
}

var client Logger

func InitMongo(c *mongo.Client) error {
	if c == nil {
		return fmt.Errorf("client is null")
	}
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := c.Ping(pingCtx, nil); err != nil {
		return fmt.Errorf("server is unpingable")
	}
	client = &mongoLogger{Db: c.Database("audit")}
	return nil
}

type mongoLogger struct {
	Db *mongo.Database
}

func (c *mongoLogger) Log(logEntry LogEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	logEntry.Timestamp = time.Now()
	res, err := c.Db.Collection(string(logEntry.EntityType)).InsertOne(ctx, logEntry)
	if err != nil {
		logrus.Warnf("failed to insert auditLog: %s", err)
		return fmt.Errorf("audit log insert failed: %w", err)
	}
	logrus.Debugf("inserted auditLog[%v]", res.InsertedID)
	return nil
}

func (c *mongoLogger) GetByEntity(entityId string, entityType EntityType, cursor time.Time, limit int64) (LogEntries, error) {
	findTimeout := 3 * time.Second
	findCtx, cancelFind := context.WithTimeout(context.Background(), findTimeout)
	defer cancelFind()
	res, err := c.Db.Collection(string(entityType)).
		Find(findCtx, bson.M{"entityId": entityId, "timestamp": bson.M{"$lte": cursor}}, options.Find().SetLimit(limit))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("timeout[%v] on find", findTimeout)
		}
		return nil, fmt.Errorf("find failed: %w", err)
	}
	defer res.Close(findCtx)

	var results LogEntries
	decodeTimeout := 3 * time.Second
	decodeCtx, cancelDecode := context.WithTimeout(context.Background(), decodeTimeout)
	defer cancelDecode()
	if err := res.All(decodeCtx, &results); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("timeout[%v] on decode", decodeTimeout)
		}
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	return results, nil
}
