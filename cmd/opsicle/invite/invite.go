package invite

import (
	"opsicle/cmd/opsicle/invite/team"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(team.Command)
}

var Command = &cobra.Command{
	Use:     "invite",
	Aliases: []string{"i"},
	Short:   "Adds users to group-based resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
