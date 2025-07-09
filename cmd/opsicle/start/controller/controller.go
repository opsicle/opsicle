package controller

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"strings"

	"github.com/spf13/cobra"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "fs-storage-path",
		Short:        'p',
		DefaultValue: "./.opsicle",
		Usage:        "specifies the path to a directory where Opsicle data resides",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "db-host",
		Short:        'H',
		DefaultValue: "localhost:5432",
		Usage:        "specifies the hostname (including port) of the database",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "db-name",
		Short:        'N',
		DefaultValue: "opsicle",
		Usage:        "specifies the name of the central database schema",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "db-user",
		Short:        'U',
		DefaultValue: "opsicle",
		Usage:        "specifies the username to use to login",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "db-password",
		Short:        'P',
		DefaultValue: "opsicle",
		Usage:        "specifies the password to use to login",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "storage-mode",
		Short:        's',
		DefaultValue: common.StorageFilesystem,
		Usage:        fmt.Sprintf("specifies what type of storage we are using, one of ['%s']", strings.Join(common.Storages, "'")),
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "controller",
	Aliases: []string{"c"},
	Short:   "Starts the controller component",
	Long:    "Starts the controller component which serves as the API layer that user interfaces can connect to to perform actions",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
