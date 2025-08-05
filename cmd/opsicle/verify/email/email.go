package email

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tea "github.com/charmbracelet/bubbletea"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "code",
		DefaultValue: "",
		Usage:        "the verification code sent to your email",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "controller-url",
		DefaultValue: "http://localhost:54321",
		Usage:        "defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "email",
	Short: "Verifies your email address",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, err := controller.GetSessionToken()
		if err == nil {
			return fmt.Errorf("looks like you're already logged in, run `opsicle logout` first before running this command")
		}

		inputCode := viper.GetString("code")

		fmt.Printf("Welcome to\n%s\n", cli.Logo)
		if inputCode == "" {
			fmt.Printf("To get started, we'll need a couple of details from you:\n\n")
		}

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
					Id:          "code",
					Placeholder: "Your verification code",
					Type:        cli.PromptString,
					Value:       viper.GetString("code"),
				},
			},
		})
		prompt := tea.NewProgram(model)
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("failed to get user input: %s", err)
		}
		if model.GetExitCode() == cli.PromptCancelled {
			fmt.Println("Wrong command? It happens; well, see you again!")
			return nil
		}

		code := model.GetValue("code")

		controllerUrl := viper.GetString("controller-url")
		logrus.Debugf("creating new client...")
		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			Id:            "opsicle/initialize/user",
		})
		if err != nil {
			return fmt.Errorf("failed to create controller client: %s", err)
		}

		logrus.Debugf("sending request to server...")
		verifyUserV1Output, err := client.VerifyUserV1(controller.VerifyUserV1Input{
			Code: code,
		})
		if err != nil {
			fmt.Printf("We couldn't verify the code that you have sent :/\n")
			logrus.Debugf("failed to verify user: %s", err)
			return fmt.Errorf("failed to verify user")
		}

		logrus.Debugf("successfully verified user[%s]", verifyUserV1Output.Email)

		fmt.Printf("Thank you for verifying your email address!\n")
		fmt.Printf("Next steps:\n")
		fmt.Printf("  1. Run `opsicle login` to get started\n")
		return nil
	},
}
