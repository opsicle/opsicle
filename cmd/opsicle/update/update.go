package update

import (
	"opsicle/cmd/opsicle/update/org"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(org.Command)
}

var Command = &cobra.Command{
	Use:     "update",
	Aliases: []string{"patch", "p", "u"},
	Short:   "Updates resources and stuff",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
