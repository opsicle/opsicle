package org

import (
	"opsicle/cmd/opsicle/get/org/token"
	"opsicle/internal/cli"
	"opsicle/internal/config"

	"github.com/spf13/cobra"
)

var flags cli.Flags = cli.Flags{}.Append(config.GetControllerUrlFlags())

func init() {
	Command.AddCommand(token.Command.Get())
}

var Command = cli.NewCommand(cli.CommandOpts{
	Flags:   flags,
	Use:     "org",
	Aliases: []string{"o"},
	Short:   "Retrieves data related to an organisation",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		return cmd.Help()
	},
})
