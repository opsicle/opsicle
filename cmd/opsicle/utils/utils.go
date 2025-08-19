package utils

import (
	"opsicle/cmd/opsicle/utils/get"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(get.Command)
}

var Command = &cobra.Command{
	Use:   "utils",
	Short: "Utility scripts to help with debugging",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
