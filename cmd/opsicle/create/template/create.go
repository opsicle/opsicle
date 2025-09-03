package template

import (
	"fmt"
	"opsicle/internal/automations"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/pkg/controller"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "controller-url",
		DefaultValue: "http://localhost:54321",
		Usage:        "Defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "file",
		Short:        'f',
		DefaultValue: "",
		Usage:        "Path to an Automation Template",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "template",
	Aliases: []string{"automation-template", "at", "tmpl", "t"},
	Short:   "Creates an automation template in Opsicle",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/list/audit-logs"
		sessionToken, err := cli.RequireAuth(controllerUrl, methodId)
		if err != nil {
			fmt.Println("⚠️  You must be logged-in to run this command")
			return err
		}

		automationTemplatePath := viper.GetString("file")
		if automationTemplatePath == "" {
			cli.PrintBoxedErrorMessage(
				"Specify the path to an automation template using --file",
			)
			return fmt.Errorf("automation template file path undefined")
		}
		absolutePathToAutomationTemplate, err := common.ToAbsolutePath(automationTemplatePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path of file: %w", err)
		}
		logrus.Debugf("loading automation template from path[%s]", absolutePathToAutomationTemplate)
		automationTemplateData, err := os.ReadFile(absolutePathToAutomationTemplate)
		if err != nil {
			return fmt.Errorf("failed to load automation template from path[%s]: %w", absolutePathToAutomationTemplate, err)
		}
		var automationTemplateInstance automations.Template
		if err := yaml.Unmarshal(automationTemplateData, &automationTemplateInstance); err != nil {
			return fmt.Errorf("failed to validate automation template: %w", err)
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

		automationTemplate, err := client.CreateAutomationTemplateV1(controller.CreateAutomationTemplateV1Input{
			Data: automationTemplateData,
		})
		if err != nil {
			return fmt.Errorf("automation template creation failed: %w", err)
		}

		fmt.Printf("✅ AutomationTemplate['%s'] has been created with UUID '%s'\n", automationTemplateInstance.Metadata.Name, automationTemplate.Data.Id)

		return nil
	},
}
