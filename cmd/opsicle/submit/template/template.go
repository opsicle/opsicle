package template

import (
	"opsicle/cmd/opsicle/create/template"

	"github.com/spf13/cobra"
)

var flags = template.Flags

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     template.Command.Use,
	Aliases: template.Command.Aliases,
	Short:   template.Command.Short + " (alias of `opsicle create template`)",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: template.Command.RunE,
}
