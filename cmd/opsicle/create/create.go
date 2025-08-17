package create

import (
	"opsicle/cmd/opsicle/create/approval_request"
	"opsicle/cmd/opsicle/create/mfa"
	"opsicle/cmd/opsicle/create/org"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(approval_request.Command)
	Command.AddCommand(mfa.Command)
	Command.AddCommand(org.Command)
}

var Command = &cobra.Command{
	Use:     "create",
	Aliases: []string{"create", "a", "c", "+"},
	Short:   "Creates resources in Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
