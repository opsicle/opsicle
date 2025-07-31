package initialize

import (
	"opsicle/cmd/opsicle/initialize/controller"
	"opsicle/cmd/opsicle/initialize/user"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(controller.Command)
	Command.AddCommand(user.Command)
}

var Command = &cobra.Command{
	Use:     "initialize",
	Aliases: []string{"init", "i"},
	Short:   "Initialises Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
