package login

import (
	"errors"
	"fmt"
	"opsicle/internal/auth"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"
	"os"

	"github.com/sirupsen/logrus"
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
		Name:         "email",
		DefaultValue: "",
		Usage:        "The email address to login to Opsicle with",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mfa-token",
		DefaultValue: "",
		Usage:        "If specified, the command will automatically include this token in your login attempt",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "password",
		DefaultValue: "",
		Usage:        "The password for your account to be used with your email address to authenticate",
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
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/create/org/user"
		_, err := cli.RequireAuth(controllerUrl, methodId)
		if err == nil {
			fmt.Println("⚠️ You're already logged in, run `opsicle logout` first before running this command")
			return fmt.Errorf("you are already logged in")
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
			return fmt.Errorf("failed to get user input: %w", err)
		}
		if model.GetExitCode() == cli.PromptCancelled {
			fmt.Println("See you soon maybe?")
			return errors.New("user cancelled action")
		}

		email := model.GetValue("email")
		if _, err := auth.IsEmailValid(email); err != nil {
			fmt.Printf("⚠️  The provided email (%s) was not valid\n", email)
			return fmt.Errorf("email invalid")
		}
		password := model.GetValue("password")

		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			Id:            "opsicle/login",
		})
		if err != nil {
			return fmt.Errorf("failed to create controller client: %w", err)
		}
		hostname, _ := os.Hostname()

		createSessionInput := controller.CreateSessionV1Input{
			Email:    email,
			Password: password,
			Hostname: hostname,
		}

		createSessionOutput, err := client.CreateSessionV1(createSessionInput)
		if err != nil {
			switch true {
			case errors.Is(err, controller.ErrorConnectionTimedOut):
				fmt.Println("⚠️  Seems like the controller service is under heavy load, try again later?")
				return fmt.Errorf("request timed out")

			case errors.Is(err, controller.ErrorConnectionRefused):
				fmt.Println("⚠️  Seems like the controller service is offline")
				return fmt.Errorf("controller unreachable")

			case errors.Is(err, controller.ErrorEmailUnverified):
				fmt.Println("⚠️  Verify your email first using `opsicle verify email`")
				return fmt.Errorf("email has not been verified")

			case errors.Is(err, controller.ErrorInvalidCredentials):
				fmt.Println("⚠️  The provided credentials doesn't seem correct, try again")
				return fmt.Errorf("credentials validation failed")

			case errors.Is(err, controller.ErrorMfaRequired):
				fmt.Println("💡 We've detected that MFA is enabled on your account, please enter your MFA token:")
				loginId := *createSessionOutput.Data.LoginId
				mfaType := *createSessionOutput.Data.MfaType
				fmt.Printf("%s: %s\n", mfaType, loginId)
				switch mfaType {
				case controller.MfaTypeTotp:
					mfaModel := cli.CreatePrompt(cli.PromptOpts{
						Buttons: []cli.PromptButton{
							{
								Label: "Submit",
								Type:  cli.PromptButtonSubmit,
							},
							{
								Label: "Cancel / Ctrl + C",
								Type:  cli.PromptButtonCancel,
							},
						},
						Inputs: []cli.PromptInput{
							{
								Id:          "mfa-token",
								Placeholder: "Your MFA Token",
								Type:        cli.PromptString,
								Value:       viper.GetString("mfa-token"),
							},
						},
					})
					mfaPrompt := tea.NewProgram(mfaModel)
					if _, err := mfaPrompt.Run(); err != nil {
						return fmt.Errorf("failed to get user input: %w", err)
					}
					if mfaModel.GetExitCode() == cli.PromptCancelled {
						fmt.Println("😥 We couldn't get an MFA code from you")
						return errors.New("user cancelled action")
					}
					mfaToken := mfaModel.GetValue("mfa-token")
					createSessionOutput, err = client.StartSessionWithMfaV1(controller.StartSessionWithMfaV1Input{
						Hostname: hostname,
						LoginId:  loginId,
						MfaType:  mfaType,
						MfaToken: mfaToken,
					})
					if err != nil {
						fmt.Println("⚠️ Unfortunately, we couldn't authenticate you")
						if errors.Is(err, controller.ErrorMfaTokenInvalid) {
							return errors.New("invalid mfa token")
						}
						logrus.Debugf("failed to start session: %s", err)
						return errors.New("failed to start session")
					}

				default:
					return fmt.Errorf("failed to create session for unexpected reasons: %w", err)
				}
			default:
				return fmt.Errorf("failed to create session for unexpected reasons: %w", err)
			}
		}

		sessionFilePath, err := controller.GetSessionTokenPath()
		if err != nil {
			logrus.Debugf("failed to get session token path: %s", err)
			fmt.Println("⚠️ We couldn't get a path to store your session token, your session was not created")
			return fmt.Errorf("failed to get session token path")
		}
		sessionTokenData := []byte(createSessionOutput.Data.SessionToken)
		if len(sessionTokenData) == 0 {
			fmt.Println("⚠️ Looks like something went very wrong, your session was not created")
			return fmt.Errorf("unexpected server response")
		}
		if err := os.WriteFile(sessionFilePath, sessionTokenData, 0600); err != nil {
			logrus.Debugf("failed to write session token to path[%s]: %s", sessionFilePath, err)
			fmt.Printf("⚠️ We couldn't write your session token to path[%s]\n", sessionFilePath)
			return fmt.Errorf("session token write failed")
		}

		fmt.Printf("👋 Welcome back!\nCurrent session ID: %s\n", createSessionOutput.Data.SessionId)
		return nil
	},
}
