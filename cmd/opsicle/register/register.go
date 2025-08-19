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
		var client *controller.Client
		var err error

		controllerUrl := viper.GetString("controller-url")

		sessionToken, sessionFilePath, err := controller.GetSessionToken()
		if err == nil {
			client, err = controller.NewClient(controller.NewClientOpts{
				BearerAuth: &controller.NewClientBearerAuthOpts{
					Token: sessionToken,
				},
				ControllerUrl: controllerUrl,
				Id:            "opsicle/register",
			})
			if err != nil {
				return fmt.Errorf("failed to create controller client: %s", err)
			}
			if _, err := client.ValidateSessionV1(); err != nil {
				if errors.Is(err, controller.ErrorAuthRequired) || errors.Is(err, controller.ErrorSessionExpired) {
					if err := controller.DeleteSessionToken(); err != nil {
						return fmt.Errorf("failed to delete session token at `%s`: %s\ndo it yourself before re-running this command:\n```\nrm -rf %s\n```", sessionFilePath, err, sessionFilePath)
					}
					fmt.Println("💬 You've been automatically logged out of your previous session that is no longer valid")
				}
			}
		}

		logrus.Debugf("creating new client...")
		client, err = controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			Id:            "opsicle/register",
		})
		if err != nil {
			return fmt.Errorf("failed to create controller client: %s", err)
		}

		inputEmail := viper.GetString("email")
		inputPassword := viper.GetString("password")
		if inputPassword != "" {
			fmt.Println(
				"⚠️ !!! WARNING !!! ⚠️\n" +
					"Using a password directly on the command line isn't generally recommended\n" +
					"since anyone can see it using the `history` command. Run `history -c` to\n" +
					"remove this from this shell if this is a shared shell")
			fmt.Println("")
		}

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
		if _, err := auth.IsEmailValid(email); err != nil {
			authErrors = append(authErrors, err)
		}
		if _, err := auth.IsPasswordValid(password); err != nil {
			authErrors = append(authErrors, err)
		}
		if len(authErrors) > 0 {
			allAuthErrors := errors.Join(authErrors...)
			fmt.Println("⛔️ Your credentials are invalid")
			if errors.Is(allAuthErrors, auth.ErrorPasswordNoUppercase) {
				fmt.Println("  > Password requires at least 1 uppercase character [A-Z]")
			}
			if errors.Is(allAuthErrors, auth.ErrorPasswordNoLowercase) {
				fmt.Println("  > Password requires at least 1 lower character [a-z]")
			}
			if errors.Is(allAuthErrors, auth.ErrorPasswordNoNumber) {
				fmt.Println("  > Password requires at least 1 numeric character [0-9]")
			}
			if errors.Is(allAuthErrors, auth.ErrorPasswordNoSymbol) {
				fmt.Println("  > Password requires at least 1 symbol character")
			}
			if errors.Is(allAuthErrors, auth.ErrorEmailAliasesNotAllowed) {
				fmt.Println("  > Email aliases are not allowed")
			}
			if errors.Is(allAuthErrors, auth.ErrorEmailDomainInvalid) {
				fmt.Println("  > Email domain is invalid")
			}
			if errors.Is(allAuthErrors, auth.ErrorEmailDomainTldNotAllowlisted) {
				fmt.Println("  > Email domain TLD is not allowlisted")
			}
			if errors.Is(allAuthErrors, auth.ErrorEmailEmptyDomain) {
				fmt.Println("  > Email address has no domain")
			}
			if errors.Is(allAuthErrors, auth.ErrorEmailInvalidAt) {
				fmt.Println("  > Email address has either no '@' or more than 1 '@' symbol")
			}
			if errors.Is(allAuthErrors, auth.ErrorEmailUserPartConsecutiveSymbols) {
				fmt.Println("  > Email address cannot contain more than 1 consecutive synbol")
			}
			if errors.Is(allAuthErrors, auth.ErrorEmailUserPartInvalidLength) {
				fmt.Println("  > Email user-part cannot be empty or longer than 64 characters")
			}
			if errors.Is(allAuthErrors, auth.ErrorEmailUserPartIlledgalChar) {
				fmt.Println("  > Email user-part contains an illegal chracter")
			}
			if errors.Is(allAuthErrors, auth.ErrorEmailUserPartLeadingSymbols) {
				fmt.Println("  > Email user-part cannot start with a symbol")
			}
			if errors.Is(allAuthErrors, auth.ErrorEmailUserPartTrailingSymbols) {
				fmt.Println("  > Email user-part cannot end with a symbol")
			}

			return fmt.Errorf("invalid credentials")
		}

		logrus.Debugf("sending request to server...")
		createUserV1Output, err := client.CreateUserV1(controller.CreateUserV1Input{
			Email:    email,
			Password: password,
		})
		if err != nil {
			fmt.Println("⛔️ Seems like that email address is invalid/not available")
			return fmt.Errorf("failed to create user")
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
