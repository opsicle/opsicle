package controller

import "opsicle/internal/cli"

var flags cli.Flags = cli.Flags{
	{
		Name:         "listen-addr",
		DefaultValue: "0.0.0.0:54321",
		Usage:        "specifies the listen address of the server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-host",
		DefaultValue: "127.0.0.1",
		Usage:        "Specifies the hostname of the MongoDB instance",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-port",
		DefaultValue: "27017",
		Usage:        "Specifies the port which the MongoDB instance is listening on",
		Type:         cli.FlagTypeString,
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
		Name:         "mysql-host",
		DefaultValue: "127.0.0.1",
		Usage:        "specifies the hostname of the database",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-port",
		DefaultValue: "3306",
		Usage:        "specifies the port which the database is listening on",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-database",
		DefaultValue: "opsicle",
		Usage:        "specifies the name of the central database schema",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-user",
		DefaultValue: "opsicle",
		Usage:        "specifies the username to use to login",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-password",
		DefaultValue: "password",
		Usage:        "specifies the password to use to login",
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
		Name:         "public-server-url",
		DefaultValue: "",
		Usage:        "specifies a url where the controller server can be accessed via - required for emails to work properly",
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
	{
		Name:         "sender-email",
		DefaultValue: "noreply@notification.opsicle.io",
		Usage:        "defines the notification sender's address",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "sender-name",
		DefaultValue: "Opsicle Notifications",
		Usage:        "defines the notification sender's name",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "session-signing-token",
		DefaultValue: "super_secret_session_signing_token",
		Usage:        "specifies the token used to sign sessions",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "smtp-username",
		DefaultValue: "noreply@notification.opsicle.io",
		Usage:        "defines the smtp server user's email address",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "smtp-password",
		DefaultValue: "",
		Usage:        "defines the smtp server user's password",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "smtp-hostname",
		DefaultValue: "smtp.eu.mailgun.org",
		Usage:        "defines the smtp server's hostname",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "smtp-port",
		DefaultValue: 587,
		Usage:        "defines the smtp server's port",
		Type:         cli.FlagTypeInteger,
	},
}
