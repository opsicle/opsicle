package verify

import (
	"opsicle/cmd/opsicle/verify/email"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(email.Command)
}

var Command = &cobra.Command{
	Use:   "verify",
	Short: "Verifies your account on Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
