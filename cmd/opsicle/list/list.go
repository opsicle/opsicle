package list

import (
	"opsicle/cmd/opsicle/list/approval_request"
	"opsicle/cmd/opsicle/list/orgs"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(approval_request.Command)
	Command.AddCommand(orgs.Command)
}

var Command = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "Lists resources in Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
