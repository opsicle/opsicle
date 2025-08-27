package remove

import (
	"opsicle/cmd/opsicle/remove/org"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(org.Command)
}

var Command = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"delete", "del", "rm"},
	Short:   "Removes resources and stuff",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
