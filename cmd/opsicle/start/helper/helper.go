package helper

import (
	"opsicle/cmd/opsicle/start/helper/telegrambot"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(telegrambot.Command)
}

var Command = &cobra.Command{
	Use:     "helper",
	Aliases: []string{"h"},
	Short:   "Runs a selection of helper tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
