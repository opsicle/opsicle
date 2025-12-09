package queue

import (
	"context"
	"time"
)

var instance Instance

type Type string

const (
	TypeNats Type = "qt_nats"
)

var queueMap = map[string]Instance{}

func Register(id string, client Instance) {
	queueMap[id] = client
}

func Get() Instance {
	return instance
}

type Instance interface {
	Push(PushOpts) (*PushOutput, error)
	Pop(PopOpts) (*Message, error)
	Subscribe(SubscribeOpts) error
}

type Message struct {
	Data    []byte `json:"data"`
	Subject string `json:"subject"`
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
