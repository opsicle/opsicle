package coordinator

import (
	"opsicle/internal/cli"
	"opsicle/internal/config"
)

var flags = cli.Flags{}.
	Append(config.GetListenAddrFlags(13372)).
	Append(config.GetMongoFlags()).
	Append(config.GetNatsFlags()).
	Append(config.GetRedisFlags())
