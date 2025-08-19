package start

import (
	"opsicle/cmd/opsicle/utils/start/httpreceiver"
	"opsicle/cmd/opsicle/utils/start/telegrambot"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(httpreceiver.Command)
	Command.AddCommand(telegrambot.Command)
}

var Command = &cobra.Command{
	Use:   "start",
	Short: "Starts stuff",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
