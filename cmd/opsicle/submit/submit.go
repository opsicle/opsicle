package submit

import (
	"opsicle/cmd/opsicle/submit/template"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(template.Command)
}

var Command = &cobra.Command{
	Use:     "submit",
	Aliases: []string{"sub", "sm"},
	Short:   "Submits resources to Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
