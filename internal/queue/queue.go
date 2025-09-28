package queue

import (
	"context"
	"fmt"
	"time"
)

type Type string

const (
	TypeNats Type = "qt_nats"
)

var queueMap = map[string]Queue{}

func Register(id string, client Queue) {
	queueMap[id] = client
}

func Get(id string) (Queue, error) {
	queueInstance, queueIdValid := queueMap[id]
	if !queueIdValid {
		return nil, fmt.Errorf("no such queue")
	}
	return queueInstance, nil
}

type Queue interface {
	Close() error
	Connect() error
	Push(PushOpts) (*PushOutput, error)
	Pop(PopOpts) (*Message, error)
	Subscribe(SubscribeOpts) error
}

type Message struct {
	Data    []byte
	Subject string
}

type MessageHandler func(context.Context, Message) error

type PopOpts struct {
	ConsumerId string
	Queue      QueueOpts
}

type PushOpts struct {
	Data   []byte
	Queue  QueueOpts
	Stream *StreamOpts
}

type PushOutput struct {
	MessageSizeBytes int
	Queue            QueueOpts
}

type QueueOpts struct {
	Stream  string
	Subject string
}

type SubscribeOpts struct {
	ConsumerId string
	Context    context.Context
	Handler    MessageHandler
	Queue      QueueOpts
	Stream     *StreamOpts
	NakBackoff time.Duration
}

type StreamOpts struct {
	MaxMessagesCount int64
	MaxSizeBytes     int64
	ReplicaCount     int
}
