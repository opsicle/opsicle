package initialize

import (
	"opsicle/cmd/opsicle/admin/initialize/controller"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(controller.Command)
}

var Command = &cobra.Command{
	Use:     "initialize",
	Aliases: []string{"init", "i"},
	Short:   "Initialises Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
