package register

import (
	"errors"
	"fmt"
	"opsicle/internal/auth"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tea "github.com/charmbracelet/bubbletea"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "email",
		DefaultValue: "",
		Usage:        "the email address you are signing up to Opsicle with",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "password",
		DefaultValue: "",
		Usage:        "the password for your account to be used with your email address to authenticate",
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
	Use:   "register",
	Short: "Register for an Opsicle platform account",
	Long:  "Register for an account on an Opsicle instance. If you are using your own instance, set the value of the --controller-url flag to your controller's address (this address needs to be reachable from your host machine)",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, err := controller.GetSessionToken()
		if err == nil {
			return fmt.Errorf("looks like you're already logged in, run `opsicle logout` first before running this command")
		}

		inputEmail := viper.GetString("email")
		inputPassword := viper.GetString("password")

		fmt.Printf("Welcome to\n%s\n", cli.Logo)
		if inputEmail == "" || inputPassword == "" {
			fmt.Printf("To get started, we'll need a couple of details from you:\n\n")
		} else {
			fmt.Printf("We see you've already specified your credentials via the CLI flags, we'll process it now...\n\n")
		}

		model := cli.CreatePrompt(cli.PromptOpts{
			Buttons: []cli.PromptButton{
				{
					Label: "Register",
					Type:  cli.PromptButtonSubmit,
				},
				{
					Label: "Cancel / Ctrl + C",
					Type:  cli.PromptButtonCancel,
				},
			},
			Inputs: []cli.PromptInput{
				{
					Id:          "email",
					Placeholder: "Your email address",
					Type:        cli.PromptString,
					Value:       viper.GetString("email"),
				},
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
			return errors.New("thank you for your interest in Opsicle, we hope to see you on our platform")
		}

		email := model.GetValue("email")
		password := model.GetValue("password")

		authErrors := []error{}
		if !auth.IsEmailValid(email) {
			authErrors = append(authErrors, fmt.Errorf("failed to get a valid email, received '%s'", email))
		}
		if _, err := auth.IsPasswordValid(password); err != nil {
			authErrors = append(authErrors, fmt.Errorf("failed to get a valid password: %s", err))
		}
		if len(authErrors) > 0 {
			return fmt.Errorf("failed to validate credentials:\n%s", errors.Join(authErrors...))
		}

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
		createUserV1Output, err := client.CreateUserV1(controller.CreateUserV1Input{
			Email:    email,
			Password: password,
		})
		if err != nil {
			return fmt.Errorf("failed to create user: %s", err)
		}

		logrus.Debugf("successfully created user[%s] with id[%s]", createUserV1Output.Data.Email, createUserV1Output.Data.Id)

		fmt.Printf("We've successfully created an account using the email '%s'\n", createUserV1Output.Data.Email)
		fmt.Printf("Next steps:\n")
		fmt.Printf("  1. Check your inbox for a verification code\n")
		fmt.Printf("  2. Run `opsicle verify email`\n")
		fmt.Printf("  3. Paste your verification code when prompted\n")
		fmt.Printf("  4. Enjoy Opsicle!\n")
		return nil
	},
}
