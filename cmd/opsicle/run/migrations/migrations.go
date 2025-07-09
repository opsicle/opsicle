package migrations

import (
	"opsicle/internal/cli"

	"github.com/spf13/cobra"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "migrations-path",
		Short:        'p',
		DefaultValue: "./migrations",
		Usage:        "specifies the path to the database migrations",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "migrations",
	Short: "Runs any database migrations",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
