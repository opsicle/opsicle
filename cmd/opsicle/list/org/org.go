package org

import (
	"opsicle/cmd/opsicle/list/org/tokens"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(tokens.Command)
}

var Command = &cobra.Command{
	Use:     "org",
	Aliases: []string{"organization", "organisation"},
	Short:   "Lists organisation resources in Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
