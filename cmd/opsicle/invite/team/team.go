package team

import (
	"opsicle/cmd/opsicle/create/org/user"

	"github.com/spf13/cobra"
)

func init() {
	user.Flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "team-member",
	Aliases: []string{"team", "t"},
	Short:   "Invite a team member to an organisation (alias of `opsicle create org user)",
	PreRun: func(cmd *cobra.Command, args []string) {
		user.Flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		rootCmd := cmd.Root()
		rootCmd.SetArgs([]string{"create", "org", "user"})
		_, err := rootCmd.ExecuteC()
		return err
	},
}
