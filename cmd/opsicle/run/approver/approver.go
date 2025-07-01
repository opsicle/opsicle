package approver

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	currentFlag := "approver"
	Command.PersistentFlags().StringP(
		currentFlag,
		"a",
		"",
		"specifies the approver service to use",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
}

var Command = &cobra.Command{
	Use:   "approver",
	Short: "Runs an approval manifest",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
