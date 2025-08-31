package check

import (
	"errors"
	"fmt"
	"opsicle/cmd/opsicle/check/audit_database"
	"opsicle/cmd/opsicle/check/cache"
	"opsicle/cmd/opsicle/check/database"
	"opsicle/cmd/opsicle/check/queue"
	"opsicle/internal/cli"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "all",
		DefaultValue: false,
		Usage:        "When this is defined, all checks are ran",
		Type:         cli.FlagTypeBool,
	},
}

func init() {
	Command.AddCommand(audit_database.Command)
	Command.AddCommand(cache.Command)
	Command.AddCommand(database.Command)
	Command.AddCommand(queue.Command)

	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "check",
	Short: "Runs checks on resources",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if viper.GetBool("all") {
			errs := []error{}
			erroredCommands := []string{}
			commandNames := []string{}
			for _, command := range cmd.Commands() {
				commandNames = append(commandNames, command.Name())
			}
			for _, commandName := range commandNames {
				rootCmd := cmd.Root()
				rootCmd.SetArgs([]string{"check", commandName})
				_, err := rootCmd.ExecuteC()
				if err != nil {
					errs = append(errs, err)
					erroredCommands = append(erroredCommands, commandName)
				}
			}
			if len(errs) > 0 {
				cli.PrintBoxedErrorMessage(fmt.Sprintf(
					"You may be experiencing reduced functionality!\n\n"+
						"The following checks failed:\n- %s",
					strings.Join(erroredCommands, "\n- "),
				))
				return errors.Join(errs...)
			}
			cli.PrintBoxedSuccessMessage("All checks passed")
			return nil
		}
		return cmd.Help()
	},
}
