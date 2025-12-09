package org

import (
	"errors"
	"fmt"
	"opsicle/cmd/opsicle/create/org/template"
	"opsicle/cmd/opsicle/create/org/token"
	"opsicle/cmd/opsicle/create/org/user"
	"opsicle/internal/cli"
	"opsicle/internal/config"
	"opsicle/internal/types"
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
}.Append(config.GetControllerUrlFlags())

func init() {
	Command.AddCommand(user.Command)
	Command.AddCommand(template.Command)
	Command.AddCommand(token.Command)
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

		controllerUrl := viper.GetString("controller-url")
		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			BearerAuth: &controller.NewClientBearerAuthOpts{
				Token: sessionToken,
			},
			Id: "opsicle/create/org",
		})
		if err != nil {
			return fmt.Errorf("failed to create controller client: %w", err)
		}

		output, err := client.ValidateSessionV1()
		if err != nil || output.Data.IsExpired {
			if err := controller.DeleteSessionToken(); err != nil {
				fmt.Printf("⚠️ We failed to remove the session token for you, please do it yourself\n")
			}
			fmt.Println("⚠️  Please login again using `opsicle login`")
			return fmt.Errorf("session invalid")
		}

		fmt.Println("✨ Let's create an organisation\n" +
			"\n" +
			"🪧  Organisation display name\n" +
			"  - Minimum 6 characters\n" +
			"  - Maximum 255 characters\n" +
			"  - Cannot start with a symbol\n" +
			"  - Cannot end with a symbol\n" +
			"🔑 Organisation codeword\n" +
			"  - Minimum 6 characters\n" +
			"  - Maximum 32 characters\n" +
			"  - Only alphanumeric characters\n" +
			"  - No special characters asides from '-' is allowed\n" +
			"  - Cannot start or end with '-'\n" +
			"  - Will be normalised to lowercase",
		)
		fmt.Println("")

		orgCode := viper.GetString("org-code")
		orgName := viper.GetString("org-name")

	askUser:
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
					Value:       orgName,
				},
				{
					Id:          "org-code",
					Placeholder: "Organisation codeword",
					Type:        cli.PromptString,
					Value:       orgCode,
				},
			},
		})
		prompt := tea.NewProgram(model)
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("failed to get user input: %w", err)
		}
		if model.GetExitCode() == cli.PromptCancelled {
			return errors.New("user cancelled")
		}
		orgCode = strings.ToLower(model.GetValue("org-code"))
		orgName = model.GetValue("org-name")

		validationErrors := []error{}
		errorMessages := []string{}
		if err := validate.OrgCode(orgCode); err != nil {
			if errors.Is(err, validate.ErrorStringTooShort) {
				errorMessages = append(errorMessages, "Organisation code word is too short")
			}
			if errors.Is(err, validate.ErrorStringTooLong) {
				errorMessages = append(errorMessages, "Organisation code word is too long")
			}
			if errors.Is(err, validate.ErrorNotLatinAlnum) {
				errorMessages = append(errorMessages, "Organisation code word has a non latin alphanumeric character")
			}
			if errors.Is(err, validate.ErrorPrefixedWithNonLatinAlnum) {
				errorMessages = append(errorMessages, "Organisation code word cannot start with a non alphanumeric character")
			}
			if errors.Is(err, validate.ErrorPostfixedWithNonLatinAlnum) {
				errorMessages = append(errorMessages, "Organisation code word cannot end with a non alphanumeric character")
			}
			validationErrors = append(validationErrors, fmt.Errorf("failed to validate org code: %w", err))
		}
		orgName = strings.Trim(orgName, " _-")
		if err := validate.OrgName(orgName); err != nil {
			if errors.Is(err, validate.ErrorStringTooShort) {
				errorMessages = append(errorMessages, "Organisation display name is too short")
			}
			if errors.Is(err, validate.ErrorStringTooLong) {
				errorMessages = append(errorMessages, "Organisation display name is too long")
			}
			if errors.Is(err, validate.ErrorNotInAllowlistedCharacters) {
				errorMessages = append(errorMessages, "Organisation display name has an invalid character")
			}
			validationErrors = append(validationErrors, fmt.Errorf("failed to validate org name: %w", err))
		}
		if len(validationErrors) > 0 {
			fmt.Printf("❌ We couldn't validate your input:\n- %s\n", strings.Join(errorMessages, "\n- "))
			logrus.Debugf("failed to validate input: %s", validationErrors)
			return fmt.Errorf("input validation failed")
		}

		createOrgOutput, err := client.CreateOrgV1(controller.CreateOrgV1Input{
			Name: orgName,
			Code: orgCode,
		})
		if err != nil {
			if errors.Is(err, types.ErrorOrgExists) {
				fmt.Printf("🙇 The organisation codeword[%s] is already in use, please pick another:\n\n", orgCode)
				orgCode = ""
				goto askUser
			}
			return fmt.Errorf("failed to create org: %w", err)
		}

		cli.PrintBoxedSuccessMessage(
			fmt.Sprintf(
				"Organisation '%s' with codeword '%s' has been successfully created",
				createOrgOutput.Data.Id,
				createOrgOutput.Data.Code,
			),
		)

		return nil
	},
}
