package update

import (
	"opsicle/cmd/opsicle/update/org"
	"opsicle/cmd/opsicle/update/template"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(org.Command)
	Command.AddCommand(template.Command)
}

var Command = &cobra.Command{
	Use:     "update",
	Aliases: []string{"patch", "p", "u"},
	Short:   "Updates resources and stuff",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
