package add

import (
	"opsicle/cmd/opsicle/add/template"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(template.Command)
}

var Command = &cobra.Command{
	Use:     "add",
	Aliases: []string{"create", "a", "c"},
	Short:   "Adds/creates resources in Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
