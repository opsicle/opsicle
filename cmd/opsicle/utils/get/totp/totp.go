package totp

import (
	"fmt"
	"opsicle/internal/auth"
	"opsicle/internal/cli"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "seed",
		DefaultValue: "",
		Usage:        "the seed to use for generating a totp token",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "totp",
	Short: "Generates a TOTP seed and ten minutes worth of TOTP tokens",
	Long:  "Generates a TOTP seed and ten minutes worth of TOTP tokens. If no seed is specified, generates a TOTP seed",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		seed := viper.GetString("seed")
		if seed == "" {
			var err error
			seed, err = auth.CreateTotpSeed("issuer", "account")
			if err != nil {
				return fmt.Errorf("failed to create totp seed: %w", err)
			}
			logrus.Infof("generated the following totp seed")
			fmt.Println(seed)
		}
		validity := 10 * time.Minute
		codes, err := auth.CreateTotpTokens(seed, validity)
		if err != nil {
			return fmt.Errorf("failed to create totp tokens: %w", err)
		}
		logrus.Infof("generated the following totp codes:")
		for _, code := range codes {
			fmt.Println(code)
		}
		return nil
	},
}
