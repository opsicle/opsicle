package print

import (
	"opsicle/internal/cli"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "confirmation",
		DefaultValue: false,
		Usage:        "Show a confirmation modal",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "info",
		DefaultValue: false,
		Usage:        "Show an information modal",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "warn",
		DefaultValue: false,
		Usage:        "Show a warning modal",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "error",
		DefaultValue: false,
		Usage:        "Show an error modal",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "success",
		DefaultValue: false,
		Usage:        "Show a success modal",
		Type:         cli.FlagTypeBool,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "print",
	Aliases: []string{"p"},
	Short:   "Prints text using in-built CLI UI components",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch true {
		case viper.GetBool("warn") && viper.GetBool("confirmation"):
			cli.ShowWarningWithConfirmation(strings.Join(args, " "), false)
		case viper.GetBool("confirmation"):
			cli.ShowConfirmation(cli.ShowConfirmationOpts{
				Title:   "Confirm?",
				Message: strings.Join(args, " "),
			})
		case viper.GetBool("warn"):
			cli.PrintBoxedWarningMessage(
				strings.Join(args, " "),
			)
		case viper.GetBool("error"):
			cli.PrintBoxedErrorMessage(
				strings.Join(args, " "),
			)
		case viper.GetBool("success"):
			cli.PrintBoxedSuccessMessage(
				strings.Join(args, " "),
			)
		case viper.GetBool("info"):
			fallthrough
		default:
			cli.PrintBoxedInfoMessage(
				strings.Join(args, " "),
			)
		}
		return nil
	},
}
