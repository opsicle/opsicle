package admin

import (
	"opsicle/cmd/opsicle/admin/create"
	"opsicle/cmd/opsicle/admin/initialize"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(create.Command)
	Command.AddCommand(initialize.Command)
}

var Command = &cobra.Command{
	Use:   "admin",
	Short: "Privileged management commands for Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
