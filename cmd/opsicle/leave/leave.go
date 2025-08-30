package leave

import (
	"opsicle/cmd/opsicle/leave/org"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(org.Command)
}

var Command = &cobra.Command{
	Use:     "leave",
	Aliases: []string{"l"},
	Short:   "Leaves group-based resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
