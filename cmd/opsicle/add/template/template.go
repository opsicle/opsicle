package template

import (
	"github.com/spf13/cobra"
)

func init() {
}

var Command = &cobra.Command{
	Use:   "template <template-path>",
	Short: "Adds an automation template",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
