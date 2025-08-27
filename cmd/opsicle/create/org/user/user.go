package user

import (
	"errors"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/validate"
	"opsicle/pkg/controller"

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
		Name:         "email",
		DefaultValue: "",
		Usage:        "The email address to invite to your organisation",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "org",
		DefaultValue: "",
		Usage:        "Codeword for the organisation to add someone to",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "user",
	Aliases: []string{"o"},
	Short:   "Adds a user to an organisation",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/create/org/user"
		sessionToken, err := cli.RequireAuth(controllerUrl, methodId)
		if err != nil {
			return err
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

		orgCode := viper.GetString("org")
		if orgCode == "" {
			logrus.Debugf("retrieving available organisations to current user")
			listOrgsOutput, err := client.ListOrgsV1()
			if err != nil {
				return fmt.Errorf("controller request failed")
			}

			fmt.Println("✨ Select an organisation to add users to:")
			fmt.Println("")
			choices := []cli.SelectorChoice{}
			for _, org := range listOrgsOutput.Data {
				choices = append(choices, cli.SelectorChoice{
					Description: org.Name,
					Label:       org.Code,
					Value:       org.Code,
				})
			}
			orgSelection := cli.CreateSelector(cli.SelectorOpts{
				Choices: choices,
			})
			orgSelector := tea.NewProgram(orgSelection)
			if _, err := orgSelector.Run(); err != nil {
				return fmt.Errorf("failed to get user input: %w", err)
			}
			if orgSelection.GetExitCode() == cli.PromptCancelled {
				return errors.New("user cancelled")
			}
			orgCode = orgSelection.GetValue()
		}
		org, err := client.GetOrgV1(controller.GetOrgV1Input{
			Code: orgCode,
		})
		if err != nil {
			return fmt.Errorf("org retrieval failed: %w", err)
		}

		if viper.GetString("email") == "" {
			fmt.Printf("✨ Enter the email of the person you want to add to %s:\n\n", org.Data.Name)
		}

	askForUserEmail:
		userEmailInput := cli.CreatePrompt(cli.PromptOpts{
			Buttons: []cli.PromptButton{
				{
					Label: "Add User",
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
					Placeholder: "Invitee's email address",
					Type:        cli.PromptString,
					Value:       viper.GetString("email"),
				},
			},
		})
		userEmailPrompt := tea.NewProgram(userEmailInput)
		if _, err := userEmailPrompt.Run(); err != nil {
			return fmt.Errorf("failed to get user input: %w", err)
		}
		if userEmailInput.GetExitCode() == cli.PromptCancelled {
			fmt.Println("😥 We couldn't get an MFA code from you")
			return errors.New("user cancelled action")
		}
		userEmail := userEmailInput.GetValue("email")

		if err := validate.Email(userEmail); err != nil {
			fmt.Printf("⚠️  The email '%s' doesn't look valid, try again:\n\n", userEmail)
			goto askForUserEmail
		}

		fmt.Printf("⏳ Adding user['%s'] to org['%s']...\n", userEmail, org.Data.Name)
		fmt.Println("")

		createOrgUserOutput, err := client.CreateOrgUserV1(controller.CreateOrgUserV1Input{
			Email:                 userEmail,
			OrgId:                 org.Data.Id,
			IsTriggerEmailEnabled: true,
		})
		if err != nil {
			if errors.Is(err, controller.ErrorInvitationExists) {
				fmt.Printf("⚠️  Looks like an invitation for the user '%s' already exists\n", userEmail)
				return fmt.Errorf("invitation already exists")
			}
			if errors.Is(err, controller.ErrorUserExistsInOrg) {
				fmt.Printf("⚠️  Looks like the user '%s' already exists in your org\n", userEmail)
				return fmt.Errorf("user already in org")
			}
			return fmt.Errorf("failed to add user: %w", err)
		}

		if createOrgUserOutput.Data.IsExistingUser {
			fmt.Println("✅ User has been invited and an email code has been sent to their email address")
		} else {
			fmt.Println("ℹ️  The specified user is not on Opsicle, but we've sent them the join code which they can use to join your organisation once they've signed up")
		}

		return nil
	},
}
