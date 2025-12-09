package check

import (
	"opsicle/cmd/opsicle/utils/check/audit_database"
	"opsicle/cmd/opsicle/utils/check/cache"
	"opsicle/cmd/opsicle/utils/check/certificate"
	"opsicle/cmd/opsicle/utils/check/database"
	"opsicle/cmd/opsicle/utils/check/queue"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(certificate.Command)
	Command.AddCommand(audit_database.Command.Get())
	Command.AddCommand(cache.Command.Get())
	Command.AddCommand(database.Command.Get())
	Command.AddCommand(queue.Command.Get())
}

var Command = &cobra.Command{
	Use:   "check",
	Short: "Checks stuff",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
