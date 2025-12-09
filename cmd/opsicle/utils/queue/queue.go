package queue

import (
	"opsicle/cmd/opsicle/utils/queue/pop"
	"opsicle/cmd/opsicle/utils/queue/push"
	"opsicle/cmd/opsicle/utils/queue/subscribe"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(push.Command.Get())
	Command.AddCommand(pop.Command.Get())
	Command.AddCommand(subscribe.Command.Get())
}

var Command = &cobra.Command{
	Use:     "queue",
	Aliases: []string{"q"},
	Short:   "Utility functions to interact with the queue",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
