package org

import (
	"errors"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/validate"
	"opsicle/pkg/controller"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "controller-url",
		Short:        'u',
		DefaultValue: "http://localhost:54321",
		Usage:        "defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "org-code",
		DefaultValue: "",
		Usage:        "code of the orgnaisation to be created",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "org-name",
		DefaultValue: "",
		Usage:        "name of the orgnaisation to be created",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "org",
	Aliases: []string{"organisation", "organization", "o"},
	Short:   "Creates a new organisation",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionToken, _, err := controller.GetSessionToken()
		if err != nil {
			fmt.Println("⚠️ You must be logged-in to run this command")
			return fmt.Errorf("login is required")
		}

		fmt.Println("✨ Awesome, let's create an organisation")
		fmt.Println("")
		fmt.Println("Let's get some details about your organisation:")
		fmt.Println("🪧  Organisation display name")
		fmt.Println("  - Minimum 6 characters")
		fmt.Println("  - Maximum 255 characters")
		fmt.Println("  - Cannot start with a symbol")
		fmt.Println("  - Cannot end with a symbol")
		fmt.Println("🔑 Organisation codeword")
		fmt.Println("  - Minimum 6 characters")
		fmt.Println("  - Maximum 32 characters")
		fmt.Println("  - Only alphanumeric characters")
		fmt.Println("  - No special characters asides from '-' is allowed")
		fmt.Println("  - Cannot start o end with '-'")
		fmt.Println("  - Will be normalised to lowercase")
		fmt.Println("")
		model := cli.CreatePrompt(cli.PromptOpts{
			Buttons: []cli.PromptButton{
				{
					Label: "Create Org",
					Type:  cli.PromptButtonSubmit,
				},
				{
					Label: "Cancel / Ctrl + C",
					Type:  cli.PromptButtonCancel,
				},
			},
			Inputs: []cli.PromptInput{
				{
					Id:          "org-name",
					Placeholder: "Organisation display name",
					Type:        cli.PromptString,
					Value:       viper.GetString("org-name"),
				},
				{
					Id:          "org-code",
					Placeholder: "Organisation codeword",
					Type:        cli.PromptString,
					Value:       viper.GetString("org-code"),
				},
			},
		})
		prompt := tea.NewProgram(model)
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("failed to get user input: %s", err)
		}
		if model.GetExitCode() == cli.PromptCancelled {
			return errors.New("user cancelled")
		}
		orgCode := model.GetValue("org-code")
		orgName := model.GetValue("org-name")

		validationErrors := []error{}
		errorMessages := []string{}
		if err := validate.OrgCode(orgCode); err != nil {
			if errors.Is(err, validate.ErrorTooShort) {
				errorMessages = append(errorMessages, "Organisation code word is too short")
			}
			if errors.Is(err, validate.ErrorTooLong) {
				errorMessages = append(errorMessages, "Organisation code word is too long")
			}
			if errors.Is(err, validate.ErrorInvalidPrefixCharacter) {
				errorMessages = append(errorMessages, "Organisation code word has an invalid prefix character")
			}
			if errors.Is(err, validate.ErrorInvalidPostfixCharacter) {
				errorMessages = append(errorMessages, "Organisation code word has an invalid postfix character")
			}
			if errors.Is(err, validate.ErrorInvalidCharacter) {
				errorMessages = append(errorMessages, "Organisation code word has an invalid character")
			}
			validationErrors = append(validationErrors, fmt.Errorf("failed to validate org code: %w", err))
		}
		orgName = strings.Trim(orgName, " _-")
		if err := validate.OrgName(orgName); err != nil {
			if errors.Is(err, validate.ErrorTooShort) {
				errorMessages = append(errorMessages, "Organisation display name is too short")
			}
			if errors.Is(err, validate.ErrorTooLong) {
				errorMessages = append(errorMessages, "Organisation display name is too long")
			}
			if errors.Is(err, validate.ErrorInvalidCharacter) {
				errorMessages = append(errorMessages, "Organisation display name has an invalid character")
			}
			validationErrors = append(validationErrors, fmt.Errorf("failed to validate org name: %w", err))
		}
		if validationErrors != nil {
			fmt.Printf("❌ We couldn't validate your input:\n- %s\n", strings.Join(errorMessages, "\n- "))
			logrus.Debugf("failed to validate input: %s", validationErrors)
			return fmt.Errorf("input validation failed")
		}

		fmt.Println(sessionToken)

		// controllerUrl := viper.GetString("controller-url")
		// client, err := controller.NewClient(controller.NewClientOpts{
		// 	ControllerUrl: controllerUrl,
		// 	BearerAuth: &controller.NewClientBearerAuthOpts{
		// 		Token: sessionToken,
		// 	},
		// 	Id: "opsicle/create/org",
		// })
		// if err != nil {
		// 	return fmt.Errorf("failed to create controller client: %s", err)
		// }

		return nil
	},
}
