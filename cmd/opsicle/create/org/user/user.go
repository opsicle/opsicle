package user

import (
	"opsicle/cmd/opsicle/invite/team"

	"github.com/spf13/cobra"
)

func init() {
	team.Flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "user",
	Aliases: []string{"u"},
	Short:   "Invite a team member to an organisation (alias of `opsicle invite team`)",
	RunE: func(cmd *cobra.Command, args []string) error {
		rootCmd := cmd.Root()
		rootCmd.SetArgs([]string{"invite", "team"})
		_, err := rootCmd.ExecuteC()
		return err
	},
}
