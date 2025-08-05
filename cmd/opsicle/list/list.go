package list

import (
	"opsicle/cmd/opsicle/list/approval_request"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(approval_request.Command)
}

var Command = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "Retrieves lists of resources in Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
