package can

import (
	"opsicle/cmd/opsicle/can/org"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(org.Command)
}

var Command = &cobra.Command{
	Use:   "can",
	Short: "Evaluates permissions in Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
