package controller

import (
	"opsicle/internal/cli"
	"opsicle/internal/config"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "api-keys",
		DefaultValue: []string{},
		Usage:        "specifies an API key for accessing protected endpoints",
		Type:         cli.FlagTypeStringSlice,
	},
	{
		Name:         "public-server-url",
		DefaultValue: "",
		Usage:        "specifies a url where the controller server can be accessed via - required for emails to work properly",
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
}.
	Append(config.GetListenAddrFlags(13371)).
	Append(config.GetMongoFlags()).
	Append(config.GetMysqlFlags()).
	Append(config.GetNatsFlags()).
	Append(config.GetRedisFlags()).
	Append(config.GetSmtpFlags())
