package coordinator

import (
	"opsicle/internal/cli"
	"opsicle/internal/config"
)

var flags = cli.Flags{
	{
		Name:         "controller-api-key",
		DefaultValue: "",
		Usage:        "defines the API key to use to communicate with the controller component",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "controller-url",
		DefaultValue: "http://localhost:13371",
		Usage:        "defines the url which the controller is available at",
		Type:         cli.FlagTypeString,
	},
}.
	Append(config.GetListenAddrFlags(13372)).
	Append(config.GetMongoFlags()).
	Append(config.GetNatsFlags()).
	Append(config.GetRedisFlags())
