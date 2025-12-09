package config

import "opsicle/internal/cli"

const (
	MysqlHost     = "mysql-host"
	MysqlPort     = "mysql-port"
	MysqlDatabase = "mysql-database"
	MysqlUsername = "mysql-username"
	MysqlPassword = "mysql-password"
)

func GetMysqlFlags() cli.Flags {
	return cli.Flags{
		{
			Name:         MysqlHost,
			DefaultValue: "127.0.0.1",
			Usage:        "specifies the hostname of the database",
			Type:         cli.FlagTypeString,
		},
		{
			Name:         MysqlPort,
			DefaultValue: "3306",
			Usage:        "specifies the port which the database is listening on",
			Type:         cli.FlagTypeString,
		},
		{
			Name:         MysqlDatabase,
			DefaultValue: "opsicle",
			Usage:        "specifies the name of the central database schema",
			Type:         cli.FlagTypeString,
		},
		{
			Name:         MysqlUsername,
			DefaultValue: "opsicle",
			Usage:        "specifies the username to use to login",
			Type:         cli.FlagTypeString,
		},
		{
			Name:         MysqlPassword,
			DefaultValue: "password",
			Usage:        "specifies the password to use to login",
			Type:         cli.FlagTypeString,
		},
	}
}
