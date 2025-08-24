package join

import (
	"opsicle/cmd/opsicle/join/org"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(org.Command)
}

var Command = &cobra.Command{
	Use:     "join",
	Aliases: []string{"j"},
	Short:   "Join group-based resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
