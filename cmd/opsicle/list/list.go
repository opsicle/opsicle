package list

import (
	"opsicle/cmd/opsicle/list/approval_request"
	"opsicle/cmd/opsicle/list/audit_logs"
	"opsicle/cmd/opsicle/list/org"
	"opsicle/cmd/opsicle/list/orgs"
	"opsicle/cmd/opsicle/list/templates"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(approval_request.Command)
	Command.AddCommand(audit_logs.Command)
	Command.AddCommand(org.Command)
	Command.AddCommand(orgs.Command)
	Command.AddCommand(templates.Command)
}

var Command = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "Lists resources in Opsicle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
