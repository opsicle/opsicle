package totpgenerator

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const cmdCtx = "o-start-helper-totpgenerator-"

func init() {
	currentFlag := "seed"
	Command.Flags().String(
		currentFlag,
		"",
		"the seed to use for generating a totp token",
	)
	viper.BindPFlag(cmdCtx+currentFlag, Command.Flags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)
}

var Command = &cobra.Command{
	Use:     "totpgenerator",
	Aliases: []string{"tgbot", "tg"},
	Short:   "Generates a TOTP seed and ten minutes worth of TOTP tokens",
	Long:    "Generates a TOTP seed and ten minutes worth of TOTP tokens. If no seed is specified, generates a TOTP seed",
	RunE: func(cmd *cobra.Command, args []string) error {

		return cmd.Help()
	},
}
