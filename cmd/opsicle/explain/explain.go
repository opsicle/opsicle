package explain

import (
	"opsicle/cmd/opsicle/explain/exitcode"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(exitcode.Command)
}

var Command = &cobra.Command{
	Use:     "explain",
	Aliases: []string{"ex"},
	Short:   "Explains things",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
