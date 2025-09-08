package template

import (
	"opsicle/cmd/opsicle/update/template/version"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(version.Command)
}

var Command = &cobra.Command{
	Use:     "template",
	Aliases: []string{"tmpl", "t"},
	Short:   "Updates automation templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
