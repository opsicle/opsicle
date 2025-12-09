package template

import (
	"errors"
	"fmt"
	"opsicle/internal/automations"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/config"
	"opsicle/pkg/controller"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var Flags cli.Flags = cli.Flags{
	{
		Name:         "file",
		Short:        'f',
		DefaultValue: "",
		Usage:        "Path to an Automation Template",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "org",
		DefaultValue: "",
		Usage:        "Define the organization that this template should be uploaded to (prompted if omitted)",
		Type:         cli.FlagTypeString,
	},
}.Append(config.GetControllerUrlFlags())

func init() {
	Flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "template",
	Aliases: []string{"automation-template", "at", "tmpl", "t"},
	Short:   "Creates an automation template in Opsicle",
	PreRun: func(cmd *cobra.Command, args []string) {
		Flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/submit/template"
	enforceAuth:
		sessionToken, err := cli.RequireAuth(controllerUrl, methodId)
		if err != nil {
			rootCmd := cmd.Root()
			rootCmd.SetArgs([]string{"login"})
			_, err := rootCmd.ExecuteC()
			if err != nil {
				return err
			}
			goto enforceAuth
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
		fmt.Printf("⏳ Loading template from path[%s]\n", absolutePathToAutomationTemplate)
		automationTemplateData, err := os.ReadFile(absolutePathToAutomationTemplate)
		if err != nil {
			return fmt.Errorf("failed to load automation template from path[%s]: %w", absolutePathToAutomationTemplate, err)
		}
		var templateInstance automations.Template
		if err := yaml.Unmarshal(automationTemplateData, &templateInstance); err != nil {
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

		orgIdentifier := strings.TrimSpace(viper.GetString("org"))
		listOrgsOutput, err := client.ListOrgsV1()
		if err != nil {
			return fmt.Errorf("failed to list organizations: %w", err)
		}
		if len(listOrgsOutput.Data) == 0 {
			cli.PrintBoxedErrorMessage("You are not a member of any organization. Create or join one before submitting templates.")
			return fmt.Errorf("no organizations available")
		}

		orgsById := map[string]controller.ListOrgsV1OutputDataOrg{}
		orgsByCode := map[string]controller.ListOrgsV1OutputDataOrg{}
		for _, org := range listOrgsOutput.Data {
			orgCopy := org
			orgsById[orgCopy.Id] = orgCopy
			orgsByCode[strings.ToLower(orgCopy.Code)] = orgCopy
		}

		var selectedOrg controller.ListOrgsV1OutputDataOrg
		isOrgSpecified := orgIdentifier != ""
		if isOrgSpecified {
			if org, ok := orgsById[orgIdentifier]; ok {
				selectedOrg = org
			} else if org, ok := orgsByCode[strings.ToLower(orgIdentifier)]; ok {
				selectedOrg = org
			} else {
				fmt.Printf("⚠️  Unable to find organization matching '%s'. Please select one from the list below.\n\n", orgIdentifier)
			}
		}

		if selectedOrg.Id == "" {
			fmt.Println("Select the organization to upload this template to:")
			selectorChoices := make([]cli.SelectorChoice, 0, len(listOrgsOutput.Data))
			for _, org := range listOrgsOutput.Data {
				selectorChoices = append(selectorChoices, cli.SelectorChoice{
					Description: fmt.Sprintf("Code: %s | Type: %s", org.Code, org.Type),
					Label:       org.Name,
					Value:       org.Id,
				})
			}
			orgSelector := cli.CreateSelector(cli.SelectorOpts{Choices: selectorChoices})
			selectorProgram := tea.NewProgram(orgSelector)
			if _, err := selectorProgram.Run(); err != nil {
				return fmt.Errorf("failed to get user input: %w", err)
			}
			if orgSelector.GetExitCode() == cli.PromptCancelled {
				return errors.New("user cancelled")
			}
			selectedOrgId := orgSelector.GetValue()
			org, ok := orgsById[selectedOrgId]
			if !ok {
				return fmt.Errorf("selected organization could not be resolved")
			}
			selectedOrg = org
		}

		if selectedOrg.Id == "" {
			return fmt.Errorf("organization selection failed")
		}

		fmt.Printf("⏳ Submitting template with name[%s] to organization[%s]...\n", templateInstance.GetName(), selectedOrg.Name)
		automationTemplate, err := client.SubmitOrgTemplateV1(controller.SubmitOrgTemplateV1Input{
			OrgId: selectedOrg.Id,
			Data:  automationTemplateData,
		})
		if err != nil {
			return fmt.Errorf("automation template creation failed: %w", err)
		}

		if automationTemplate.Data.Version > 1 {
			fmt.Printf("✅ Your automation template <%s> has been updated to v%v\n", automationTemplate.Data.Name, automationTemplate.Data.Version)
		} else {
			fmt.Printf("✅ A new automation template <%s> has been created at v%v with UUID '%s'\n", automationTemplate.Data.Name, automationTemplate.Data.Version, automationTemplate.Data.Id)
		}

		return nil
	},
}
