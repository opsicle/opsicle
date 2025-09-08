package check

import (
	"opsicle/cmd/opsicle/utils/check/certificate"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(certificate.Command)
}

var Command = &cobra.Command{
	Use:   "check",
	Short: "Checks stuff",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
