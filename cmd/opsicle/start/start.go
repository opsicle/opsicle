package start

import (
	"opsicle/cmd/opsicle/start/approver"
	"opsicle/cmd/opsicle/start/controller"
	"opsicle/cmd/opsicle/start/coordinator"
	"opsicle/cmd/opsicle/start/reporter"
	"opsicle/cmd/opsicle/start/worker"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(approver.Command.Get())
	Command.AddCommand(controller.Command.Get())
	Command.AddCommand(coordinator.Command.Get())
	Command.AddCommand(reporter.Command)
	Command.AddCommand(worker.Command)
}

var Command = &cobra.Command{
	Use:     "start",
	Aliases: []string{"st"},
	Short:   "Starts one of Opsicle's core services",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
