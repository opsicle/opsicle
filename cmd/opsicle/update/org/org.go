package org

import (
	"opsicle/cmd/opsicle/update/org/user"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(user.Command)
}

var Command = &cobra.Command{
	Use:     "org",
	Aliases: []string{"organisation", "organization", "o"},
	Short:   "Updates organisation resources and stuff",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
