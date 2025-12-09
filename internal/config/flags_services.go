package config

import "opsicle/internal/cli"

func GetControllerUrlFlags() cli.Flags {
	return cli.Flags{
		{
			Name:         "controller-url",
			Short:        'u',
			DefaultValue: "http://localhost:13371",
			Usage:        "the url of the controller service",
			Type:         cli.FlagTypeString,
		},
	}
}
func GetCoordinatorUrlFlags() cli.Flags {
	return cli.Flags{
		{
			Name:         "coordinator-url",
			Short:        'u',
			DefaultValue: "http://localhost:13372",
			Usage:        "the url of the coordinator service",
			Type:         cli.FlagTypeString,
		},
	}
}
