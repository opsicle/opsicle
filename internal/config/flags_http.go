package config

import (
	"fmt"
	"opsicle/internal/cli"
)

func GetListenAddrFlags(port int) cli.Flags {
	return cli.Flags{
		{
			Name:         "listen-addr",
			DefaultValue: fmt.Sprintf("0.0.0.0:%v", port),
			Usage:        "specifies the listen address of the server",
			Type:         cli.FlagTypeString,
		},
	}
}
