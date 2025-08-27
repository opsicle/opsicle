package org

import (
	"opsicle/cmd/opsicle/remove/org/user"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(user.Command)
}

var Command = &cobra.Command{
	Use:     "org",
	Aliases: []string{"organisation", "organization", "o"},
	Short:   "Removes resources from an org",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
