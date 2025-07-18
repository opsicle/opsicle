package create

import (
	"opsicle/cmd/opsicle/create/approval_request"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(approval_request.Command)
}

var Command = &cobra.Command{
	Use:     "create",
	Aliases: []string{"create", "a", "c", "+"},
	Short:   "Creates/creates resources in Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
