package start

import (
	"opsicle/cmd/opsicle/start/approver"
	"opsicle/cmd/opsicle/start/controller"
	"opsicle/cmd/opsicle/start/helper"
	"opsicle/cmd/opsicle/start/worker"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(approver.Command)
	Command.AddCommand(controller.Command)
	Command.AddCommand(helper.Command)
	Command.AddCommand(worker.Command)
}

var Command = &cobra.Command{
	Use:     "start",
	Aliases: []string{"s"},
	Short:   "Starts one of Opsicle's core services",
	GroupID: "app",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
