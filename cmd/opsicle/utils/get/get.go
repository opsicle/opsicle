package get

import (
	"opsicle/cmd/opsicle/utils/get/totp"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(totp.Command)
}

var Command = &cobra.Command{
	Use:     "get",
	Aliases: []string{"g"},
	Short:   "Retrieves stuff",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
