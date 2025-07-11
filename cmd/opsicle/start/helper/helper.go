package helper

import (
	"opsicle/cmd/opsicle/start/helper/httpreceiver"
	"opsicle/cmd/opsicle/start/helper/telegrambot"
	"opsicle/cmd/opsicle/start/helper/totpgenerator"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(httpreceiver.Command)
	Command.AddCommand(telegrambot.Command)
	Command.AddCommand(totpgenerator.Command)
}

var Command = &cobra.Command{
	Use:     "helper",
	Aliases: []string{"h"},
	Short:   "Runs a selection of helper tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
