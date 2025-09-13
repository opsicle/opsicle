package template

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "controller-url",
		Short:        'u',
		DefaultValue: "http://localhost:54321",
		Usage:        "Defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "template-name",
		Short:        't',
		DefaultValue: "",
		Usage:        "Defines the name of the template to remove",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "template",
	Aliases: []string{"tmpl", "t"},
	Short:   "Removes an automation templates from your library",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/list/templates"
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

		inputTemplateName := viper.GetString("template-name")
		templateInstance, err := cli.HandleTemplateSelection(cli.HandleTemplateSelectionOpts{
			Client:    client,
			UserInput: inputTemplateName,
			// ServiceLog: servicesLogs,
		})
		if err != nil {
			return fmt.Errorf("failed to select a template: %w", err)
		}

		warningMessage := fmt.Sprintf(
			"You are about to delete %s template[%s] created by user[%s] on %s",
			lipgloss.NewStyle().Bold(true).Render("ALL VERSIONS OF"),
			templateInstance.Name,
			*templateInstance.CreatedByEmail,
			templateInstance.CreatedAt.Local().Format("2 Jan 2006 15:04:05 -0700"),
		)
		if templateInstance.LastUpdatedByEmail != nil {
			warningMessage += fmt.Sprintf(
				" and last updated by user[%s] on %s",
				*templateInstance.LastUpdatedByEmail,
				templateInstance.LastUpdatedAt.Local().Format("2 Jan 2006 15:04:05 -0700"),
			)
		}
		warningMessage += "\n\nHit OK below to confirm"
		warningMessage += lipgloss.NewStyle().Italic(true).Render("\n\n💡 If you want to delete only a single version, use `opsicle delete template version` instead")
		if err := cli.ShowWarningWithConfirmation(
			warningMessage,
			false,
		); err != nil {
			return err
		}

		fmt.Printf("⏳ Removing template[%s] with id[%s]...\n", templateInstance.Name, templateInstance.Id)

		deleteTemplateOutput, err := client.DeleteTemplateV1(controller.DeleteTemplateV1Input{
			TemplateId: templateInstance.Id,
		})
		if err != nil {
			return fmt.Errorf("failed to delete template: %w", err)
		}
		if deleteTemplateOutput.Data.IsSuccessful {
			cli.PrintBoxedSuccessMessage(
				fmt.Sprintf("Successfully deleted automation template <%s>", deleteTemplateOutput.Data.TemplateName),
			)
		}

		return nil
	},
}
