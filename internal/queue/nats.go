package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"opsicle/internal/common"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

const (
	DefaultNatsAckWaitDuration    time.Duration = 300 * time.Second
	DefaultNatsMaxAckPendingCount int           = 64
	DefaultNatsMaxMessageCount    int64         = 1024
	DefaultNatsMaxSizeBytes       int64         = 1024 * 1024 * 128
	DefaultNatsPublishTimeout     time.Duration = 5 * time.Second
	DefaultNatsPullTimeout        time.Duration = 5 * time.Second
	DefaultNatsStreamReplicaCount int           = 1
)

func getNatsQueueInfo(opts QueueOpts) (stream, subject string) {
	stream = strings.ToLower(opts.Stream)
	subject = fmt.Sprintf("%s.%s.*", stream, strings.ToLower(opts.Subject))
	return
}

type Nats struct {
	Addr        string
	Client      *nats.Conn
	ServiceLogs chan<- common.ServiceLog

	options       []nats.Option
	streamContext nats.JetStreamContext
}

func (n *Nats) Close() error {
	if isNoopInUse {
		stopNoopServiceLog()
	}
	if err := n.Client.Drain(); err != nil {
		return fmt.Errorf("failed to drain connection[%s]: %w", n.Client.ConnectedAddr(), err)
	}
	n.Client.Close()
	return nil
}

func (n *Nats) Connect() error {
	addr := n.Addr
	var err error
	n.Client, err = nats.Connect("nats://"+addr, n.options...)
	if err != nil {
		return fmt.Errorf("failed to connect to nats: %w", err)
	}
	if !n.Client.IsConnected() {
		return fmt.Errorf("failed to verify connection")
	}
	n.streamContext, err = n.Client.JetStream()
	if err != nil {
		return fmt.Errorf("failed to get jetstream context: %w", err)
	}
	return nil
}

func (n *Nats) Push(opts PushOpts) (*PushOutput, error) {
	if err := n.ensureNats(); err != nil {
		return nil, fmt.Errorf("failed to validate nats setup: %w", err)
	}
	_, subject := getNatsQueueInfo(opts.Queue)
	ensureStreamOpts := NatsStreamOpts{
		MaxMessagesCount: DefaultNatsMaxMessageCount,
		MaxSizeBytes:     DefaultNatsMaxSizeBytes,
		Replicas:         DefaultNatsStreamReplicaCount,
		QueueInfo:        opts.Queue,
	}
	if opts.Stream != nil {
		if opts.Stream.MaxMessagesCount != 0 {
			ensureStreamOpts.MaxMessagesCount = opts.Stream.MaxMessagesCount
		}
		if opts.Stream.MaxSizeBytes != 0 {
			ensureStreamOpts.MaxSizeBytes = opts.Stream.MaxSizeBytes
		}
		if opts.Stream.ReplicaCount != 0 {
			ensureStreamOpts.Replicas = opts.Stream.ReplicaCount
		}
	}
	if err := n.ensureStream(ensureStreamOpts); err != nil {
		return nil, fmt.Errorf("failed to ensure stream: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), DefaultNatsPublishTimeout)
	defer cancel()
	_, err := n.streamContext.Publish(subject, opts.Data, nats.Context(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to publish message: %w", err)
	}
	return &PushOutput{
		MessageSizeBytes: len(opts.Data),
		Queue:            opts.Queue,
	}, nil
}

func (n *Nats) Pop(opts PopOpts) (*Message, error) {
	if err := n.ensureNats(); err != nil {
		return nil, fmt.Errorf("failed to validate nats setup: %w", err)
	}
	stream, subject := getNatsQueueInfo(opts.Queue)
	if hasMessage, err := n.hasMessages(stream, subject); err != nil {
		return nil, fmt.Errorf("failed to check for message count: %w", err)
	} else if !hasMessage {
		return nil, nil
	}
	sub, err := n.streamContext.PullSubscribe(
		subject,
		opts.ConsumerId,
		nats.BindStream(stream),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to stream[%s]: %w", stream, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), DefaultNatsPullTimeout)
	defer cancel()
	msg, err := sub.Fetch(1, nats.Context(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from subject[%s]: %w", subject, err)
	}
	if len(msg) == 0 {
		return nil, nil
	}
	msgMetadata, err := msg[0].Metadata()
	if err != nil {
		return nil, fmt.Errorf("failed to get message metadata: %w", err)
	}
	logrus.Infof("received message[%v:%v]", msgMetadata.Sequence.Consumer, msgMetadata.Sequence.Stream)
	if err := msg[0].AckSync(); err != nil {
		return nil, fmt.Errorf("failed to ack msg[%v]: %w", msgMetadata.Sequence.Stream, err)
	}
	if err := sub.Unsubscribe(); err != nil {
		return nil, fmt.Errorf("failed to set auto-unsubscribe: %w", err)
	}
	return &Message{
		Data:    msg[0].Data,
		Subject: msg[0].Sub.Subject,
	}, nil
}

type NatsSubscribeHandler func(context.Context, Message) error

// type NatsSubscribeOpts struct {
// 	ConsumerId string
// 	Context    context.Context
// 	Handler    NatsSubscribeHandler
// 	NakBackoff time.Duration
// 	QueueInfo  QueueOpts
// 	StreamOpts *StreamOpts
// }

func (n *Nats) Subscribe(opts SubscribeOpts) error {
	if err := n.ensureNats(); err != nil {
		return fmt.Errorf("failed to validate nats setup: %w", err)
	}

	stream, subject := getNatsQueueInfo(opts.Queue)
	ensureStreamOpts := NatsStreamOpts{
		MaxMessagesCount: DefaultNatsMaxMessageCount,
		MaxSizeBytes:     DefaultNatsMaxSizeBytes,
		Replicas:         DefaultNatsStreamReplicaCount,
		QueueInfo:        opts.Queue,
	}
	if opts.Stream != nil {
		if opts.Stream.MaxMessagesCount != 0 {
			ensureStreamOpts.MaxMessagesCount = opts.Stream.MaxMessagesCount
		}
		if opts.Stream.MaxSizeBytes != 0 {
			ensureStreamOpts.MaxSizeBytes = opts.Stream.MaxSizeBytes
		}
		if opts.Stream.ReplicaCount != 0 {
			ensureStreamOpts.Replicas = opts.Stream.ReplicaCount
		}
	}
	if err := n.ensureStream(ensureStreamOpts); err != nil {
		return fmt.Errorf("failed to ensure stream: %w", err)
	}

	ensureDurableOpts := NatsDurableOpts{
		Durable:    opts.ConsumerId,
		Stream:     stream,
		Subject:    subject,
		StreamOpts: ensureStreamOpts,
	}
	if err := n.ensureDurable(ensureDurableOpts); err != nil {
		return err
	}

	// --- Bind pull subscription to the durable ---
	sub, err := n.streamContext.PullSubscribe(
		subject,
		opts.ConsumerId,
		nats.Bind(stream, opts.ConsumerId),
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	n.ServiceLogs <- common.ServiceLogf(
		common.LogLevelDebug,
		"nats subscription created: "+
			"durable=%s "+
			"stream=%s "+
			"subject=%s",
		opts.ConsumerId,
		stream,
		subject,
	)

	nakBackoff := 10 * time.Second
	if opts.NakBackoff != 0 {
		nakBackoff = opts.NakBackoff
	}

	for {
		select {
		case <-opts.Context.Done():
			n.ServiceLogs <- common.ServiceLogf(
				common.LogLevelDebug,
				"nats subscription stopping: "+
					"durable=%s "+
					"stream=%s "+
					"subject=%s",
				opts.ConsumerId,
				stream,
				subject,
			)
			return opts.Context.Err()
		default:
		}

		msgs, err := sub.Fetch(1, nats.MaxWait(2*time.Second))
		if err != nil {
			// Timeout means no messages; keep polling.
			if errors.Is(err, nats.ErrTimeout) {
				continue
			}
			return fmt.Errorf("fetch: %w", err)
		}
		if len(msgs) == 0 {
			continue
		}

		msg := msgs[0]

		err = opts.Handler(opts.Context, Message{
			Data:    msg.Data,
			Subject: msg.Subject,
		})
		if err != nil {
			n.ServiceLogs <- common.ServiceLogf(
				common.LogLevelWarn,
				"🔁 nats message handling failed, sending nak with delay[%v]: %s",
				nakBackoff,
				err,
			)
			_ = msg.NakWithDelay(nakBackoff)
			continue
		}
		n.ServiceLogs <- common.ServiceLogf(
			common.LogLevelDebug,
			"✅ acking message[%s]",
			string(msg.Data),
		)
		if err := msg.Ack(); err != nil {
			return fmt.Errorf("failed to ack: %w", err)
		}
	}
}

func (n *Nats) ensureNats() error {
	errs := []error{}
	if n.Client == nil {
		errs = append(errs, ErrorClientUndefined)
	}
	if n.streamContext == nil {
		errs = append(errs, ErrorStreamingClientUndefined)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

type NatsDurableOpts struct {
	AckWait    time.Duration
	Durable    string
	Stream     string
	Subject    string
	StreamOpts NatsStreamOpts
}

func (n *Nats) ensureDurable(opts NatsDurableOpts) error {
	ci, err := n.streamContext.ConsumerInfo(opts.Stream, opts.Durable)
	if err == nil && ci != nil {
		if ci.Config.FilterSubject != opts.Subject {
			return fmt.Errorf("failed to ensure durable subject association: have=%q want=%q", ci.Config.FilterSubject, opts.Subject)
		}
		return nil
	}

	maxAck := opts.StreamOpts.MaxAckPending
	if maxAck <= 0 {
		maxAck = DefaultNatsMaxAckPendingCount
	}
	ackWait := opts.AckWait
	if ackWait <= 0 {
		ackWait = DefaultNatsAckWaitDuration
	}

	_, err = n.streamContext.AddConsumer(opts.Stream, &nats.ConsumerConfig{
		Durable:           opts.Durable,
		FilterSubject:     opts.Subject,
		AckPolicy:         nats.AckExplicitPolicy,
		AckWait:           ackWait,
		MaxAckPending:     maxAck,
		DeliverPolicy:     nats.DeliverAllPolicy,
		ReplayPolicy:      nats.ReplayInstantPolicy,
		InactiveThreshold: 0,
	})
	if err != nil && !errors.Is(err, nats.ErrConsumerNameAlreadyInUse) && !errors.Is(err, nats.ErrObjectAlreadyExists) {
		return fmt.Errorf("failed to add consumer: %w", err)
	}
	return nil
}

type NatsStreamOpts struct {
	MaxAckPending    int
	MaxMessagesCount int64
	MaxSizeBytes     int64
	Replicas         int
	StorageType      *int
	QueueInfo        QueueOpts
}

func (n *Nats) ensureStream(opts NatsStreamOpts) error {
	stream, subject := getNatsQueueInfo(opts.QueueInfo)
	if streamInfo, err := n.streamContext.StreamInfo(stream); err == nil && streamInfo != nil {
		cfg := streamInfo.Config
		if !n.isSubjectInSubjects(streamInfo.Config.Subjects, subject) {
			cfg.Subjects = append(cfg.Subjects, subject)
			if _, err := n.streamContext.UpdateStream(&cfg); err != nil {
				return fmt.Errorf("failed to update stream[%s:%s]: %w", stream, subject, err)
			}
		}
		cfg.Retention = nats.WorkQueuePolicy
		if _, err := n.streamContext.UpdateStream(&cfg); err != nil {
			return fmt.Errorf("failed to update stream retention: %w", err)
		}
		return nil
	}

	// Create new stream
	cfg := &nats.StreamConfig{
		NoAck:     false,
		Name:      stream,
		Subjects:  []string{subject},
		Replicas:  opts.Replicas,
		Retention: nats.WorkQueuePolicy,
		// Limits; -1 = unlimited
		MaxMsgs:  opts.MaxMessagesCount,
		MaxBytes: opts.MaxSizeBytes,
		Storage:  nats.FileStorage,
		Discard:  nats.DiscardOld,
	}
	if opts.StorageType != nil {
		if cfg.Storage != nats.StorageType(*opts.StorageType) {
			cfg.Storage = nats.StorageType(*opts.StorageType)
		}
	}

	if _, err := n.streamContext.AddStream(cfg); err != nil {
		return fmt.Errorf("failed to add stream[%s:%s]: %w", stream, subject, err)
	}
	return nil
}

func (n *Nats) hasMessages(stream, subject string) (bool, error) {
	type msgGetReq struct {
		LastBySubj string `json:"last_by_subj"`
	}
	type apiErr struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	}
	type apiResp struct {
		Error *apiErr `json:"error,omitempty"`
	}

	req := msgGetReq{LastBySubj: subject}
	data, _ := json.Marshal(req)

	inbox := fmt.Sprintf("$JS.API.STREAM.MSG.GET.%s", stream)
	msg, err := n.Client.Request(inbox, data, 2*time.Second)
	if err != nil {
		return false, err
	}

	var r apiResp
	if err := json.Unmarshal(msg.Data, &r); err != nil {
		return false, err
	}
	if r.Error != nil && r.Error.Code == 404 {
		return false, nil
	}
	return true, nil
}

func (n *Nats) isSubjectInSubjects(subjects []string, target string) bool {
	for _, s := range subjects {
		if s == target {
			return true
		}
	}
	return false
}
