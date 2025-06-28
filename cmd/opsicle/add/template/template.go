package template

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	currentFlag := "template-path"
	Command.PersistentFlags().StringP(
		currentFlag,
		"p",
		"",
		"specifies the path to the automation template to create",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
}

var Command = &cobra.Command{
	Use:   "template",
	Short: "Adds an automation template",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
