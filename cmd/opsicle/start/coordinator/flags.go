package coordinator

import "opsicle/internal/cli"

var flags = cli.Flags{
	{
		Name:         "listen-addr",
		DefaultValue: "0.0.0.0:11111",
		Usage:        "specifies the listen address of the server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-host",
		DefaultValue: []string{"127.0.0.1:27017"},
		Usage:        "Specifies the hostname(s) of the MongoDB instance",
		Type:         cli.FlagTypeStringSlice,
	},
	{
		Name:         "mongo-user",
		DefaultValue: "opsicle",
		Usage:        "Specifies the username to use to login to the MongoDB instance",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-password",
		DefaultValue: "password",
		Usage:        "Specifies the password to use to login to the MongoDB instance",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "nats-addr",
		DefaultValue: "localhost:4222",
		Usage:        "Specifies the hostname (including port) of the NATS server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "nats-username",
		DefaultValue: "opsicle",
		Usage:        "Specifies the username used to login to NATS",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "nats-password",
		DefaultValue: "password",
		Usage:        "Specifies the password used to login to NATS",
		Type:         cli.FlagTypeString,
	},
	{
		Name: "nats-nkey-value",
		// this default value is the development nkey, this value must be aligned
		// to the one in `./docker-compose.yml` in the root of the repository
		DefaultValue: "SUADZTA4VJHBCO7K75DQ3IN7KZGWHKEI26D2IYEABRN5TXXYHXLWNDYT4A",
		Usage:        "Specifies the nkey used to login to NATS",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "redis-addr",
		DefaultValue: "localhost:6379",
		Usage:        "defines the hostname (including port) of the redis server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "redis-username",
		DefaultValue: "opsicle",
		Usage:        "defines the username used to login to redis",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "redis-password",
		DefaultValue: "password",
		Usage:        "defines the password used to login to redis",
		Type:         cli.FlagTypeString,
	},
}
