package version

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "controller-url",
		DefaultValue: "http://localhost:54321",
		Usage:        "Defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)

}

var Command = &cobra.Command{
	Use:     "version <template-name>",
	Aliases: []string{"tmpl", "t"},
	Short:   "Sets the default version of an automation template",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/update/template/version"
		sessionToken, err := cli.RequireAuth(controllerUrl, methodId)
		if err != nil {
			fmt.Println("⚠️  You must be logged-in to run this command")
			return err
		}
		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			BearerAuth: &controller.NewClientBearerAuthOpts{
				Token: sessionToken,
			},
			Id: methodId,
		})
		if err != nil {
			return fmt.Errorf("failed to create controller client: %w", err)
		}

		templateName := ""
		isTemplateNameDefined := len(args) > 0
		if isTemplateNameDefined {
			templateName = args[0]
		} else {
			templates, err := client.ListTemplatesV1(controller.ListTemplatesV1Input{
				Limit: 20,
			})
			if err != nil {
				return fmt.Errorf("failed to list templates: %w", err)
			}
			for _, template := range templates.Data {
				fmt.Printf("%s\n", template.Name)
			}
		}
		fmt.Println(templateName)
		cli.FuzzySearch()
		return nil
	},
}
