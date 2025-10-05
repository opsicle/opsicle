package team

import (
	"errors"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/validate"
	"opsicle/pkg/controller"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Flags cli.Flags = cli.Flags{
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
	{
		Name:         "member-type",
		DefaultValue: "",
		Usage:        "Type of membership that the user should have",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	Flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "team-member",
	Aliases: []string{"team", "t"},
	Short:   "Invite a team member to an organisation (alias of `opsicle create org user)",
	PreRun: func(cmd *cobra.Command, args []string) {
		Flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/create/org/user"
	enforceAuth:
		sessionToken, err := cli.RequireAuth(controllerUrl, methodId)
		if err != nil {
			rootCmd := cmd.Root()
			rootCmd.SetArgs([]string{"login"})
			_, err := rootCmd.ExecuteC()
			if err != nil {
				return err
			}
			goto enforceAuth
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

		orgCode, err := cli.HandleOrgSelection(cli.HandleOrgSelectionOpts{
			Client:    client,
			UserInput: viper.GetString("org"),
		})
		if err != nil {
			return fmt.Errorf("org selection failed: %w", err)
		}
		org, err := client.GetOrgV1(controller.GetOrgV1Input{
			Code: *orgCode,
		})
		if err != nil {
			return fmt.Errorf("org retrieval failed: %w", err)
		}

		if viper.GetString("email") == "" {
			fmt.Printf("💬 Enter the email of the person you want to add to %s:\n\n", org.Data.Name)
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

		orgMemberTypes, err := client.ListOrgMemberTypesV1()
		if err != nil {
			return fmt.Errorf("org member types retrieval failed: %w", err)
		}
		membershipType := viper.GetString("member-type")
		isMembershipTypeSpecified := membershipType != ""
		isMembershipTypeValid := false
		if isMembershipTypeSpecified {
			for _, orgMemberType := range orgMemberTypes.Data {
				if membershipType == orgMemberType {
					isMembershipTypeValid = true
				}
			}
		}

		if !isMembershipTypeValid {
			if isMembershipTypeSpecified {
				fmt.Printf("⚠️  It looks like your indicated membership type '%s' is not valid\n", membershipType)
				fmt.Println("   Please select one from the following:")
			} else {
				fmt.Println("💬 What membership type should we grant the user?")
			}
			fmt.Println("")
			choices := []cli.SelectorChoice{}
			for _, orgMemberType := range orgMemberTypes.Data {
				choices = append(choices, cli.SelectorChoice{
					Label: orgMemberType,
					Value: orgMemberType,
				})
			}
			sort.Slice(choices, func(i, j int) bool {
				return choices[i].Label < choices[j].Label
			})
			orgMemberTypeSelection := cli.CreateSelector(cli.SelectorOpts{
				Choices: choices,
			})
			orgMemberTypeSelector := tea.NewProgram(orgMemberTypeSelection)
			if _, err := orgMemberTypeSelector.Run(); err != nil {
				return fmt.Errorf("failed to get user input: %w", err)
			}
			if orgMemberTypeSelection.GetExitCode() == cli.PromptCancelled {
				return errors.New("user cancelled")
			}
			membershipType = orgMemberTypeSelection.GetValue()
		}

		fmt.Printf("⏳ Adding <%s> to the organisation %s <%s> as a <%s>...\n", userEmail, org.Data.Name, org.Data.Code, membershipType)
		fmt.Println("")

		createOrgUserOutput, err := client.CreateOrgUserV1(controller.CreateOrgUserV1Input{
			Email:                 userEmail,
			OrgId:                 org.Data.Id,
			IsTriggerEmailEnabled: true,
			Type:                  membershipType,
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
