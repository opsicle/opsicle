package version

import (
	"errors"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sirupsen/logrus"
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
		templateId := ""
		isTemplateNameDefined := len(args) > 0
		if isTemplateNameDefined {
			templateName = args[0]
		}
		templates, err := client.ListTemplatesV1(controller.ListTemplatesV1Input{
			Limit: 20,
		})
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}
		isTemplateNameValid := false
		filterItems := []cli.FilterItem{}
		for _, template := range templates.Data {
			filterItems = append(filterItems, cli.FilterItem{
				Description: fmt.Sprintf("v%v -- %s", template.Version, template.Description),
				Label:       template.Name,
				Value:       strings.Join([]string{template.Name, template.Id}, ":"),
			})
			if isTemplateNameDefined && template.Name == templateName {
				isTemplateNameValid = true
				templateId = template.Id
				break
			}
		}
		if !isTemplateNameValid {
			sort.Slice(filterItems, func(i, j int) bool {
				return filterItems[i].Label < filterItems[j].Label
			})
			templateFiltering := cli.CreateFilter(cli.FilterOpts{
				Items: filterItems,
				Title: "💬 Which template will it be?",
			})
			templateFilter := tea.NewProgram(templateFiltering)
			if _, err := templateFilter.Run(); err != nil {
				logrus.Errorf("failed to get user input: %s", err)
				return fmt.Errorf("failed to get user input: %w", err)
			}
			if templateFiltering.GetExitCode() == cli.PromptCancelled {
				return errors.New("user cancelled")
			}
			selectedTemplate := templateFiltering.GetSelectedItem()
			templateValues := strings.Split(selectedTemplate.Value, ":")
			templateName = templateValues[0]
			templateId = templateValues[1]
		}

		fmt.Printf("💬 Loading template[%s]\n", templateName)
		logrus.Debugf("loading template[[%s] versions... ", templateId)

		templateVersionsOutput, err := client.ListTemplateVersionsV1(controller.ListTemplateVersionsV1Input{
			TemplateId: templateId,
		})
		if err != nil {
			return fmt.Errorf("failed to list template versions: %w", err)
		}
		filterItems = []cli.FilterItem{}
		for _, templateVersion := range templateVersionsOutput.Data.Versions {
			label := fmt.Sprintf("v%v by %s", templateVersion.Version, templateVersion.CreatedBy.Email)
			if templateVersion.Version == templateVersionsOutput.Data.Template.Version {
				label += " (current)"
			}
			filterItems = append(filterItems, cli.FilterItem{
				Label:       label,
				Description: fmt.Sprintf("Uploaded on %s", templateVersion.CreatedAt.Local().Format("3 Feb 2006 3:04:05 PM -0700")),
				Value:       fmt.Sprintf("%v", templateVersion.Version),
			})
		}
		templateFiltering := cli.CreateFilter(cli.FilterOpts{
			Items:              filterItems,
			Title:              "💬 Which version should be the default?",
			StartWithoutFilter: true,
		})
		templateFilter := tea.NewProgram(templateFiltering)
		if _, err := templateFilter.Run(); err != nil {
			logrus.Errorf("failed to get user input: %s", err)
			return fmt.Errorf("failed to get user input: %w", err)
		}
		if templateFiltering.GetExitCode() == cli.PromptCancelled {
			return errors.New("user cancelled")
		}
		selectedTemplate := templateFiltering.GetSelectedItem()
		selectedTemplateVersion, _ := strconv.ParseInt(selectedTemplate.Value, 10, 64)
		fmt.Printf("💬 Setting v%v as the default version...\n", selectedTemplateVersion)
		updateOutput, err := client.UpdateTemplateDefaultVersionV1(controller.UpdateTemplateDefaultVersionV1Input{
			TemplateId: templateId,
			Version:    selectedTemplateVersion,
		})
		if err != nil {
			if errors.Is(err, controller.ErrorNotFound) {
				cli.PrintBoxedErrorMessage(
					"The template you specified could not be found",
				)
				return fmt.Errorf("template not found: %w", err)
			}
			return fmt.Errorf("failed to update default template version: %w", err)
		}
		cli.PrintBoxedSuccessMessage(
			fmt.Sprintf("Updated default version of template <%s> to v%v\n", templateName, updateOutput.Data.Version),
		)

		return nil
	},
}
