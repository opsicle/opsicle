package user

import (
	"opsicle/cmd/opsicle/update/org/user/perms"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(perms.Command)
}

var Command = &cobra.Command{
	Use:     "user",
	Aliases: []string{"u"},
	Short:   "Updates organisation user",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
