package reset

import (
	"opsicle/cmd/opsicle/reset/password"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(password.Command)
}

var Command = &cobra.Command{
	Use:   "reset",
	Short: "Resets resources in Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
