package config

import "opsicle/internal/cli"

const (
	NatsAddr      = "nats-addr"
	NatsUsername  = "nats-username"
	NatsPassword  = "nats-password"
	NatsNkeyValue = "nats-nkey-value"
)

func GetNatsFlags() cli.Flags {
	return cli.Flags{
		{
			Name:         NatsAddr,
			DefaultValue: "localhost:4222",
			Usage:        "Specifies the hostname (including port) of the NATS server",
			Type:         cli.FlagTypeString,
		},
		{
			Name:         NatsUsername,
			DefaultValue: "opsicle",
			Usage:        "Specifies the username used to login to NATS",
			Type:         cli.FlagTypeString,
		},
		{
			Name:         NatsPassword,
			DefaultValue: "password",
			Usage:        "Specifies the password used to login to NATS",
			Type:         cli.FlagTypeString,
		},
		{
			Name: NatsNkeyValue,
			// this default value is the development nkey, this value must be aligned
			// to the one in `./docker-compose.yml` in the root of the repository
			DefaultValue: "SUADZTA4VJHBCO7K75DQ3IN7KZGWHKEI26D2IYEABRN5TXXYHXLWNDYT4A",
			Usage:        "Specifies the nkey used to login to NATS",
			Type:         cli.FlagTypeString,
		},
	}
}
