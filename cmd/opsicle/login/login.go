package login

import (
	"errors"
	"fmt"
	"opsicle/internal/auth"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tea "github.com/charmbracelet/bubbletea"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "controller-url",
		DefaultValue: "http://localhost:54321",
		Usage:        "Defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "org",
		DefaultValue: "",
		Usage:        "Defines the codeword of the organisation you are logging into",
		Type:         cli.FlagTypeString,
	},
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
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "login",
	Short: "Login to Opsicle",
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
		if inputPassword != "" {
			fmt.Println(
				"⚠️ !!! WARNING !!! ⚠️\n" +
					"Using a password directly on the command line isn't generally recommended\n" +
					"since anyone can see it using the `history` command. Run `history -c` to\n" +
					"remove this from this shell if this is a shared shell")
		}

		fmt.Printf("\nLogging into\n%s\n", cli.Logo)
		if inputEmail == "" || inputPassword == "" {
			fmt.Printf("To get started, we'll need a couple of details from you:\n\n")
		}

		model := cli.CreatePrompt(cli.PromptOpts{
			Buttons: []cli.PromptButton{
				{
					Label: "Login",
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
			return errors.New("See you soon maybe?")
		}

		email := model.GetValue("email")
		if !auth.IsEmailValid(email) {
			fmt.Printf("⚠️  The provided email (%s) was not valid\n", email)
			return fmt.Errorf("email invalid")
		}
		password := model.GetValue("password")
		organisation := model.GetValue("organisation")

		controllerUrl := viper.GetString("controller-url")
		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			Id:            "opsicle/login",
		})
		if err != nil {
			return fmt.Errorf("failed to create controller client: %s", err)
		}
		hostname, _ := os.Hostname()

		createSessionInput := controller.CreateSessionV1Input{
			Email:    email,
			Password: password,
			Hostname: hostname,
		}
		if organisation != "" {
			createSessionInput.OrgCode = &organisation
		}
		createSessionOutput, err := client.CreateSessionV1(createSessionInput)
		if err != nil {
			if errors.Is(err, controller.ErrorUserEmailNotVerified) {
				fmt.Println("⚠️  Verify your email first using `opsicle verify email`")
				return fmt.Errorf("email has not been verified")
			} else if errors.Is(err, controller.ErrorUserLoginFailed) {
				fmt.Println("⚠️  The provided credentials doesn't seem correct, try again")
				return fmt.Errorf("credentials validation failed")
			}
			return fmt.Errorf("failed to create session for unexpected reasons: %s", err)
		}

		sessionFilePath, err := controller.GetSessionTokenPath()
		if err != nil {
			return fmt.Errorf("failed to get session token path: %s", err)
		}
		if err := os.WriteFile(sessionFilePath, []byte(createSessionOutput.SessionToken), 0600); err != nil {
			return fmt.Errorf("failed to write session token to path[%s]: %s", sessionFilePath, err)
		}

		fmt.Printf("Welcome back!\nSession ID: %s\n", createSessionOutput.SessionId)
		return nil
	},
}
