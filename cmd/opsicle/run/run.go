package run

import (
	"opsicle/cmd/opsicle/run/automation"
	"opsicle/cmd/opsicle/run/migrations"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(automation.Command)
	Command.AddCommand(migrations.Command)
}

var Command = &cobra.Command{
	Use:     "run",
	Aliases: []string{"r"},
	Short:   "Runs scripts related to Opsicle's operation",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
