package queue

import (
	"opsicle/internal/common"
	"opsicle/internal/persistence"
)

// InitNatsOpts configures the InitNats method
type InitNatsOpts struct {
	NatsConnection *persistence.Nats
	ServiceLogs    chan<- common.ServiceLog
}

// InitNats initialises a singleton instance of a NATS queue
func InitNats(opts InitNatsOpts) error {
	instance = &Nats{
		Client:      opts.NatsConnection,
		ServiceLogs: opts.ServiceLogs,
	}
	return nil
}
