package org

import (
	"opsicle/cmd/opsicle/can/org/user"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(user.Command)
}

var Command = &cobra.Command{
	Use:   "org",
	Short: "Evaluates organisation-level permissions",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
