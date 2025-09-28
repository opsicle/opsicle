package queue

import (
	"opsicle/cmd/opsicle/utils/queue/pop"
	"opsicle/cmd/opsicle/utils/queue/push"
	"opsicle/cmd/opsicle/utils/queue/subscribe"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(push.Command)
	Command.AddCommand(pop.Command)
	Command.AddCommand(subscribe.Command)
}

var Command = &cobra.Command{
	Use:     "queue",
	Aliases: []string{"q"},
	Short:   "Utility functions to interact with the queue",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
