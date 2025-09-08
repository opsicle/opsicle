package create

import (
	"opsicle/cmd/opsicle/utils/create/certificate"
	"opsicle/cmd/opsicle/utils/create/certificate_authority"

	"github.com/spf13/cobra"
)

func init() {
	Command.AddCommand(certificate.Command)
	Command.AddCommand(certificate_authority.Command)
}

var Command = &cobra.Command{
	Use:     "create",
	Aliases: []string{"generate", "make"},
	Short:   "Creates/generates stuff",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
