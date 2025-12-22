package org

import (
	"opsicle/internal/cli"

	"github.com/spf13/cobra"
)

var Command = cli.NewCommand(cli.CommandOpts{
	Use:     "org",
	Aliases: []string{"o"},
	Short:   "Sets organisation variables in Opsicle",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		return cmd.Help()
	},
})
