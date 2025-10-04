package register

import (
	"opsicle/cmd/opsicle/create/user"
	"opsicle/cmd/opsicle/join/template"

	"github.com/spf13/cobra"
)

var flags = user.Flags

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "register",
	Aliases: template.Command.Aliases,
	Short:   template.Command.Short + " (alias of `opsicle create user`)",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		rootCmd := cmd.Root()
		rootCmd.SetArgs([]string{"create", "user"})
		return rootCmd.Execute()
	},
}
