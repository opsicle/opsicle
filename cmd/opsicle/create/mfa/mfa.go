package mfa

import (
	"errors"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"

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
	{
		Name:         "password",
		DefaultValue: "",
		Usage:        "Your password for verification (ideally, use an environment variable so the literal value doesn't show up in `history`)",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "mfa",
	Short: "Creates an MFA for the current user (you must be logged-in before running this)",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionToken, _, err := controller.GetSessionToken()
		if err != nil {
			fmt.Println("You must be logged-in to run this command")
			return fmt.Errorf("login is required")
		}

		fmt.Println("⏳ Retrieving current MFA methods...")

		controllerUrl := viper.GetString("controller-url")
		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			BearerAuth: &controller.NewClientBearerAuthOpts{
				Token: sessionToken,
			},
			Id: "opsicle/create/mfa",
		})
		if err != nil {
			return fmt.Errorf("failed to create controller client: %s", err)
		}

		// get current user mfas

		logrus.Debug("retrieving current user's registered mfa methods...")
		listUserMfasOutput, err := client.ListUserMfasV1(controller.ListUserMfasV1Input{})
		if err != nil {
			return fmt.Errorf("failed to list user mfas: %s", err)
		}
		if len(listUserMfasOutput.Data) == 0 {
			fmt.Println("💡 No MFA methods have been registered")
		}

		// get all available mfa types

		logrus.Debug("retrieving available mfa types...")
		availableMfasOutput, err := client.ListAvailableMfaTypes()
		if err != nil {
			return fmt.Errorf("failed to get available mfa methods: %s", err)
		}
		if len(availableMfasOutput.Data) == 0 {
			fmt.Println("❌ No MFA methods were enabled on the server, contact your server administrator")
			return fmt.Errorf("failed to receive any mfa method")
		}

		// check if there's any more types of mfa to create

		availableMfaMap := map[string]struct{}{}
		for _, availableMfa := range availableMfasOutput.Data {
			availableMfaMap[availableMfa.Value] = struct{}{}
		}
		for _, userMfa := range listUserMfasOutput.Data {
			delete(availableMfaMap, userMfa.Type)
		}
		if len(availableMfaMap) == 0 {
			return fmt.Errorf("no more mfa methods to register")
		}

		// mfa type selection

		fmt.Println("\n💬 Which type of MFA do you want to register?")
		fmt.Println("")

		selectorChoices := []cli.SelectorChoice{}
		for _, mfaOption := range availableMfasOutput.Data {
			selectorChoices = append(selectorChoices, cli.SelectorChoice{
				Description: mfaOption.Description,
				Label:       mfaOption.Label,
				Value:       mfaOption.Value,
			})
		}
		mfaTypeSelector := cli.CreateSelector(cli.SelectorOpts{
			Choices: selectorChoices,
		})
		selector := tea.NewProgram(mfaTypeSelector)
		if _, err := selector.Run(); err != nil {
			return fmt.Errorf("failed to get user input: %s", err)
		}
		if mfaTypeSelector.GetExitCode() == cli.PromptCancelled {
			return errors.New("user cancelled")
		}
		selectedMfaType := mfaTypeSelector.GetValue()

		fmt.Printf("✅ Awesome, let's register a new MFA of type [%s]\n", mfaTypeSelector.GetLabel())

		// password verification

		fmt.Println("\n💬 Also, we need to verify this is really you, please provide your current password:")
		fmt.Println("")

		model := cli.CreatePrompt(cli.PromptOpts{
			Buttons: []cli.PromptButton{
				{
					Label: "Verify",
					Type:  cli.PromptButtonSubmit,
				},
				{
					Label: "Cancel / Ctrl + C",
					Type:  cli.PromptButtonCancel,
				},
			},
			Inputs: []cli.PromptInput{
				{
					Id:          "password",
					Placeholder: "Your password",
					Type:        cli.PromptPassword,
					Value:       viper.GetString("password"),
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
		password := model.GetValue("password")

		fmt.Println("⏳ Hang on, we're processing your request...")

		logrus.Debugf("register mfa[%s] with password[%s]", selectedMfaType, password)

		return nil
	},
}
