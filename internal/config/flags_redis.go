package config

import "opsicle/internal/cli"

const (
	RedisAddr     = "redis-addr"
	RedisUsername = "redis-username"
	RedisPassword = "redis-password"
)

func GetRedisFlags() cli.Flags {
	return cli.Flags{
		{
			Name:         RedisAddr,
			DefaultValue: "localhost:6379",
			Usage:        "defines the hostname (including port) of the redis server",
			Type:         cli.FlagTypeString,
		},
		{
			Name:         RedisUsername,
			DefaultValue: "opsicle",
			Usage:        "defines the username used to login to redis",
			Type:         cli.FlagTypeString,
		},
		{
			Name:         RedisPassword,
			DefaultValue: "password",
			Usage:        "defines the password used to login to redis",
			Type:         cli.FlagTypeString,
		},
	}
}
