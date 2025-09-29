package create

import (
	"opsicle/cmd/opsicle/create/approval_request"
	"opsicle/cmd/opsicle/create/mfa"
	"opsicle/cmd/opsicle/create/org"
	"opsicle/cmd/opsicle/create/template"
	"opsicle/cmd/opsicle/create/user"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(approval_request.Command)
	Command.AddCommand(mfa.Command)
	Command.AddCommand(org.Command)
	Command.AddCommand(template.Command)
	Command.AddCommand(user.Command)
}

var Command = &cobra.Command{
	Use:     "create",
	Aliases: []string{"add", "a", "c"},
	Short:   "Creates/adds resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
