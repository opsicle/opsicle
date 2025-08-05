package validate

import (
	"opsicle/cmd/opsicle/validate/automationtemplate"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(automationtemplate.Command)
}

var Command = &cobra.Command{
	Use:     "validate",
	Aliases: []string{"v"},
	Short:   "Validates resource manifests",
	GroupID: "utils",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
