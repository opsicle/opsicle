package utils

import (
	"opsicle/cmd/opsicle/utils/check"
	"opsicle/cmd/opsicle/utils/create"
	"opsicle/cmd/opsicle/utils/get"
	"opsicle/cmd/opsicle/utils/print"
	"opsicle/cmd/opsicle/utils/queue"
	"opsicle/cmd/opsicle/utils/send"
	"opsicle/cmd/opsicle/utils/set"
	"opsicle/cmd/opsicle/utils/show"
	"opsicle/cmd/opsicle/utils/start"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(check.Command)
	Command.AddCommand(create.Command)
	Command.AddCommand(get.Command)
	Command.AddCommand(print.Command)
	Command.AddCommand(queue.Command)
	Command.AddCommand(send.Command)
	Command.AddCommand(set.Command)
	Command.AddCommand(show.Command)
	Command.AddCommand(start.Command)
}

var Command = &cobra.Command{
	Use:   "utils",
	Short: "Utility scripts to help with debugging",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
