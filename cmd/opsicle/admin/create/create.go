package create

import (
	"opsicle/cmd/opsicle/admin/create/user"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(user.Command)
}

var Command = &cobra.Command{
	Use:     "create",
	Aliases: []string{"c"},
	Short:   "Creates/creates resources in Opsicle via the database directly",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
