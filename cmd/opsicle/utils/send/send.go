package send

import (
	"opsicle/cmd/opsicle/utils/send/email"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(email.Command.Get())
}

var Command = &cobra.Command{
	Use:     "send",
	Aliases: []string{"s"},
	Short:   "Sends stuff",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
