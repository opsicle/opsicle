package set

import (
	"opsicle/cmd/opsicle/utils/set/cache"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(cache.Command.Get())
}

var Command = &cobra.Command{
	Use:     "set",
	Aliases: []string{"s"},
	Short:   "Retrieves stuff",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
