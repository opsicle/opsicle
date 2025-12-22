package set

import (
	"opsicle/cmd/opsicle/get/org"
	"opsicle/internal/cli"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(org.Command.Get())
}

var Command = cli.NewCommand(cli.CommandOpts{
	Use:   "set",
	Short: "Sets things in Opsicle",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		return cmd.Help()
	},
})
