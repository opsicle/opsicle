package approvals

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/config"
	"strings"

	"github.com/spf13/cobra"
)

var flags = cli.Flags{
	{
		Name:         "org",
		DefaultValue: "",
		Usage:        "codeword of the orgnaisation",
		Type:         cli.FlagTypeString,
	},
}.Append(config.GetControllerUrlFlags())

var Command = cli.NewCommand(cli.CommandOpts{
	Flags:   flags,
	Use:     "approvals <status>",
	Aliases: []string{"a"},
	Short:   "Sets the organisation approval status",
	Long: strings.Trim(`
	Sets the organisation approval <status>, when <status> is set to 'on',
	the approvals mechanism will be triggered whenever a runbook automation
	is executed. When set to 'off', no approvals are needed to trigger automations.
	Use 'opsicle add org approver' to add an approver as an organisation-wide
	approver
	`, "\n "),
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("failed to receive a valid <status>, <status> should be either 'on' or 'off'")
		}

		// TODO: implement a method that results in the column `is_approvals_enabled` of `org_config` table being set

		return nil
	},
})
