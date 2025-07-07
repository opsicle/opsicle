package migrations

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const cmdCtx = "o-run-migrations-"

func init() {
	currentFlag := "migrations-path"
	Command.PersistentFlags().StringP(
		currentFlag,
		"p",
		"./migrations",
		"specifies the path to the database migrations",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)
}

var Command = &cobra.Command{
	Use:   "migrations",
	Short: "Runs any database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
