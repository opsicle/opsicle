package config

import (
	"fmt"
	"opsicle/internal/cli"
)

const (
	MongoHosts    = "mongo-hosts"
	MongoHost     = "mongo-host"
	MongoPort     = "mongo-port"
	MongoUsername = "mongo-username"
	MongoPassword = "mongo-password"
)

func GetMongoFlags() cli.Flags {
	return cli.Flags{
		{
			Name:         MongoHosts,
			DefaultValue: []string{"127.0.0.1:27017"},
			Usage:        fmt.Sprintf("Specifies the hostname(s) of the MongoDB instance (takes precedence over flags --%s and --%s when defined)", MongoHost, MongoPort),
			Type:         cli.FlagTypeStringSlice,
		},
		{
			Name:         MongoHost,
			DefaultValue: "127.0.0.1",
			Usage:        "Specifies the hostname of the MongoDB instance",
			Type:         cli.FlagTypeString,
		},
		{
			Name:         MongoPort,
			DefaultValue: "27017",
			Usage:        "Specifies the port which the MongoDB instance is listening on",
			Type:         cli.FlagTypeString,
		},
		{
			Name:         MongoUsername,
			DefaultValue: "opsicle",
			Usage:        "Specifies the username to use to login to the MongoDB instance",
			Type:         cli.FlagTypeString,
		},
		{
			Name:         MongoPassword,
			DefaultValue: "password",
			Usage:        "Specifies the password to use to login to the MongoDB instance",
			Type:         cli.FlagTypeString,
		},
	}
}
