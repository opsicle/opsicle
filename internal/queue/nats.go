package queue

import (
	"fmt"
	"opsicle/internal/common"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

// InitNatsOpts configures the InitNats method
type InitNatsOpts struct {
	// Addr contains the hostname:port address of the NATS instance
	Addr string

	// Username defines the username to use when authenticating with NATS
	Username string

	// Password defines the password to use when authenticating with NATS
	Password string

	// NKey takes precedence over the `Username` and `Password`
	// fields; when this is specified, the standard credentials
	// are ignored in favour of using this `NKey` which is arguably
	// more secure
	NKey string

	ServiceLogs chan<- common.ServiceLog
}

// InitNats initialises a singleton instance of a NATS queue
func InitNats(opts InitNatsOpts) (*nats.Conn, error) {
	// if opts.ServiceLogs != nil {
	// 	redisOpts.ServiceLogs = &opts.ServiceLogs
	// } else {
	// 	initNoopServiceLog()
	// 	var serviceLogs chan<- common.ServiceLog = noopServiceLog
	// 	redisOpts.ServiceLogs = &serviceLogs
	// 	go startNoopServiceLog()
	// 	defer stopNoopServiceLog()
	// }
	natsOpts := []nats.Option{}
	if opts.NKey != "" {
		keyPair, err := nkeys.FromSeed([]byte(opts.NKey))
		if err != nil {
			return nil, fmt.Errorf("failed to generate keypair from nkey: %w", err)
		}
		publicKey, err := keyPair.PublicKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate public key from nkey: %w", err)
		}
		natsOpts = append(natsOpts, nats.Nkey(publicKey, keyPair.Sign))
	} else if opts.Username != "" && opts.Password != "" {
		natsOpts = append(natsOpts, nats.UserInfo(opts.Username, opts.Password))
	} else {
		return nil, fmt.Errorf("failed to receive any authentication methods")
	}
	addr := opts.Addr
	connection, err := nats.Connect("nats://"+addr, natsOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats: %w", err)
	}
	if !connection.IsConnected() {
		return nil, fmt.Errorf("failed to verify connection")
	}
	return connection, nil

}
