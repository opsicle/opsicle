package login

import (
	"opsicle/cmd/opsicle/create/session"

	"github.com/spf13/cobra"
)

var flags = session.Flags

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "login",
	Aliases: session.Command.Aliases,
	Short:   session.Command.Short + " (alias of `opsicle create session`)",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		rootCmd := cmd.Root()
		rootCmd.SetArgs([]string{"create", "session"})
		return rootCmd.Execute()
	},
}
