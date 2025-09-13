package remove

import (
	"opsicle/cmd/opsicle/remove/org"
	"opsicle/cmd/opsicle/remove/template"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(org.Command)
	Command.AddCommand(template.Command)
}

var Command = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"delete", "del", "rm"},
	Short:   "Removes resources and stuff",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
