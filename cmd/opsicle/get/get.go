package get

import (
	"opsicle/cmd/opsicle/get/approval"
	"opsicle/cmd/opsicle/get/approval_request"
	"opsicle/cmd/opsicle/get/org"
	"opsicle/cmd/opsicle/get/users"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(approval.Command)
	Command.AddCommand(approval_request.Command)
	Command.AddCommand(org.Command.Get())
	Command.AddCommand(users.Command)
}

var Command = &cobra.Command{
	Use:     "get",
	Aliases: []string{"g"},
	Short:   "Retrieves resources in Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
