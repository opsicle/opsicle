package exitcode

import (
	"fmt"
	"opsicle/internal/cli"
	"strconv"

	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:     "exitcode <exit-code>",
	Aliases: []string{"exit-code", "exit"},
	Short:   "Explains the provided exit code",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			cli.PrintBoxedErrorMessage(
				"We did not receive an exit code",
			)
			return cli.ErrorInvalidInput
		}
		exitCode, err := strconv.Atoi(args[0])
		if err != nil {
			cli.PrintBoxedErrorMessage(
				"Exit code should be a number",
			)
		}
		table := cli.Table{}
		table.Init([]string{"source", "is errored"})
		table.NewRow("generic", exitCode&cli.ExitCodeError > 0)
		table.NewRow("database issue", exitCode&cli.ExitCodeDbError > 0)
		table.NewRow("input data", exitCode&cli.ExitCodeInputError > 0)
		table.NewRow("svc unavailable", exitCode&cli.ExitCodeServiceUnavailable > 0)
		fmt.Println(table.GetString())
		return nil
	},
}
