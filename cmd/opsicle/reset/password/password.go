package password

import (
	"errors"
	"fmt"
	"opsicle/internal/auth"
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
		Name:         "email",
		DefaultValue: "",
		Usage:        "The email address you use to login to Opsicle with (applies only when you're not logged in)",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "old-password",
		DefaultValue: "",
		Usage:        "The current password for your account (applies only when you're logged in)",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "new-password",
		DefaultValue: "",
		Usage:        "The new password for your account (applies only when you're logged in)",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "password",
	Short: "Trigger this to reset your password",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		isUserLoggedIn := false
		sessionToken, _, err := controller.GetSessionToken()
		if err == nil {
			isUserLoggedIn = true
		}

		controllerClientOpts := controller.NewClientOpts{
			ControllerUrl: viper.GetString("controller-url"),
			Id:            "opsicle/reset/password",
		}
		if isUserLoggedIn {
			controllerClientOpts.BearerAuth = &controller.NewClientBearerAuthOpts{
				Token: sessionToken,
			}
		}

		client, err := controller.NewClient(controllerClientOpts)
		if err != nil {
			return fmt.Errorf("failed to initialise controller client")
		}

		if isUserLoggedIn {
			passwordRequest := cli.CreatePrompt(cli.PromptOpts{
				Buttons: []cli.PromptButton{
					{
						Label: "Update Password",
						Type:  cli.PromptButtonSubmit,
					},
					{
						Label: "Cancel / Ctrl + C",
						Type:  cli.PromptButtonCancel,
					},
				},
				Inputs: []cli.PromptInput{
					{
						Id:          "old-password",
						Placeholder: "Your current password",
						Type:        cli.PromptPassword,
						Value:       viper.GetString("old-password"),
					},
					{
						Id:          "new-password",
						Placeholder: "Your new password",
						Type:        cli.PromptPassword,
						Value:       viper.GetString("new-password"),
					},
				},
			})
			prompt := tea.NewProgram(passwordRequest)
			if _, err := prompt.Run(); err != nil {
				fmt.Println("⚠️ We didn't manage to get your current and new password")
				logrus.Debugf("failed to get password: %s", err)
				return fmt.Errorf("failed to get password")
			}
			if passwordRequest.GetExitCode() == cli.PromptCancelled {
				fmt.Println("See you soon maybe?")
				return errors.New("user cancelled action")
			}

			oldPassword := passwordRequest.GetValue("old-password")
			newPassword := passwordRequest.GetValue("new-password")

			arePasswordsValid := true
			if _, err := auth.IsPasswordValid(oldPassword); err != nil {
				arePasswordsValid = false
				fmt.Printf("⚠️ Your current password doesn't seem valid:\n%s\n", err)
			}
			if _, err := auth.IsPasswordValid(oldPassword); err != nil {
				arePasswordsValid = false
				fmt.Printf("⚠️ Your new password doesn't seem valid:\n%s\n", err)
			}
			if !arePasswordsValid {
				return fmt.Errorf("passwords are invalid")
			}

			_, err = client.ResetPasswordV1(controller.ResetPasswordV1Input{
				CurrentPassword: &oldPassword,
				NewPassword:     &newPassword,
			})
			if err != nil {
				logrus.Debugf("failed to update password: %s", err)
				if errors.Is(err, controller.ErrorInvalidCredentials) {
					fmt.Println("⚠️ Your current password doesn't seem valid")
					return fmt.Errorf("current password is invalid")
				} else if errors.Is(err, controller.ErrorInvalidInput) {
					fmt.Println("⚠️ One or both of your passwords aren't valid")
					return fmt.Errorf("one or both passwords are invalid")
				}
				return fmt.Errorf("password update failed")
			}
			fmt.Println("✅ Your password change is successful")
			return nil
		}

		fmt.Println(cli.Logo + "\n" + "😔 It happens, we understand; let us know your email and we'll get you sorted:")
		fmt.Println("")

		emailRequest := cli.CreatePrompt(cli.PromptOpts{
			Buttons: []cli.PromptButton{
				{
					Label: "Trigger Password Reset",
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
			},
		})
		prompt := tea.NewProgram(emailRequest)
		if _, err := prompt.Run(); err != nil {
			fmt.Println("⚠️ We didn't manage to get your email address to reset your password")
			logrus.Debugf("failed to get email address: %s", err)
			return fmt.Errorf("failed to get email address")
		}
		if emailRequest.GetExitCode() == cli.PromptCancelled {
			fmt.Println("See you soon maybe?")
			return errors.New("user cancelled action")
		}

		email := emailRequest.GetValue("email")
		_, err = client.ResetPasswordV1(controller.ResetPasswordV1Input{
			Email: &email,
		})
		if err != nil {
			logrus.Debugf("failed to trigger email verification for password reset: %s", err)
			return fmt.Errorf("password update failed")
		}

		fmt.Println("💡 If your email was valid, you should receive an email from us containing a verification code, enter it below along with your intended password:")
		fmt.Println("")

		verificationCodeRequest := cli.CreatePrompt(cli.PromptOpts{
			Buttons: []cli.PromptButton{
				{
					Label: "Verify & Change Password",
					Type:  cli.PromptButtonSubmit,
				},
				{
					Label: "Cancel / Ctrl + C",
					Type:  cli.PromptButtonCancel,
				},
			},
			Inputs: []cli.PromptInput{
				{
					Id:          "verification-code",
					Placeholder: "Verification code you received via email",
					Type:        cli.PromptPassword,
				},
				{
					Id:          "new-password",
					Placeholder: "Your new password",
					Type:        cli.PromptPassword,
				},
			},
		})
		prompt = tea.NewProgram(verificationCodeRequest)
		if _, err := prompt.Run(); err != nil {
			fmt.Println("⚠️ We didn't manage to get the verification code to reset your password")
			logrus.Debugf("failed to get verification code: %s", err)
			return fmt.Errorf("failed to get verification code")
		}
		if verificationCodeRequest.GetExitCode() == cli.PromptCancelled {
			fmt.Println("See you soon maybe?")
			return errors.New("user cancelled action")
		}

		verificationCode := verificationCodeRequest.GetValue("verification-code")
		newPassword := verificationCodeRequest.GetValue("new-password")
		_, err = client.ResetPasswordV1(controller.ResetPasswordV1Input{
			VerificationCode: &verificationCode,
			NewPassword:      &newPassword,
		})
		if err != nil {
			logrus.Debugf("failed to trigger password reset: %s", err)
			return fmt.Errorf("password update failed")
		}

		fmt.Println("✅ You're all set, use your ✨ shiny new password ✨ to login")

		return nil
	},
}
