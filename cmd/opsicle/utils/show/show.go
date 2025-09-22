package show

import (
	"opsicle/cmd/opsicle/utils/show/form"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(form.Command)
}

var Command = &cobra.Command{
	Use:     "show",
	Aliases: []string{"s"},
	Short:   "Shows UI components",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
