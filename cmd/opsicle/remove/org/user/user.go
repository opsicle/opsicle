package user

import (
	"errors"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/validate"
	"opsicle/pkg/controller"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
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
		Name:         "org",
		DefaultValue: "",
		Usage:        "Codeword of the organisation to list users from",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "user-id",
		DefaultValue: "",
		Usage:        "ID (or email) of the user to update",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "user",
	Aliases: []string{"account", "acc", "u"},
	Short:   "Removes a user from an organisation",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
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

		selectedOrg, err := cli.HandleOrgSelection(cli.HandleOrgSelectionOpts{
			Client: client,
			// ServiceLog: serviceLogs,
			UserInput: viper.GetString("org"),
		})
		if err != nil {
			return fmt.Errorf("org selection failed: %w", err)
		}
		org, err := client.GetOrgV1(controller.GetOrgV1Input{Ref: selectedOrg.Code})
		if err != nil {
			return fmt.Errorf("org retrieval failed: %w", err)
		}
		orgMembership, err := client.GetOrgMembershipV1(controller.GetOrgMembershipV1Input{
			OrgId: org.Data.Id,
		})
		if err != nil {
			return fmt.Errorf("org membership retrieval failed: %w", err)
		}
		if !orgMembership.Data.Permissions.CanManageUsers {
			return fmt.Errorf("not allowed")
		}

		userIdentifier := viper.GetString("user-id")
		userIdEmailMap := map[string]string{}

		orgUsers, err := client.ListOrgUsersV1(controller.ListOrgUsersV1Input{
			OrgId: org.Data.Id,
		})
		if err != nil {
			return fmt.Errorf("org users retrieval failed: %w", err)
		}
		for _, orgUser := range orgUsers.Data {
			userIdEmailMap[orgUser.UserId] = orgUser.UserEmail
			userIdEmailMap[orgUser.UserEmail] = orgUser.UserId
		}
		isUserIdentifiedSpecified := userIdentifier != ""
		isUserIdentifiedValid := false
		if userIdentifier != "" {
			if altIdentifier, ok := userIdEmailMap[userIdentifier]; ok {
				if _, err := uuid.Parse(userIdentifier); err != nil {
					userIdentifier = altIdentifier
				}
				isUserIdentifiedValid = true
			}
		}
		if !isUserIdentifiedValid {
			if isUserIdentifiedSpecified {
				fmt.Printf("⚠️  It looks like your specified user '%s' does not exist in the organisation\n", userIdentifier)
				fmt.Println("   Please select a user from your organisation to remove:")
			} else {
				fmt.Println("Select the user from your organisation that you want to remove:")
			}
			fmt.Println("")
			orgUsers, err := client.ListOrgUsersV1(controller.ListOrgUsersV1Input{
				OrgId: org.Data.Id,
			})
			if err != nil {
				return fmt.Errorf("org users retrieval failed: %w", err)
			}
			choices := []cli.SelectorChoice{}
			for _, orgUser := range orgUsers.Data {
				userIdEmailMap[orgUser.UserId] = orgUser.UserEmail
				choices = append(choices, cli.SelectorChoice{
					Description: fmt.Sprintf("currently type[%s]", orgUser.MemberType),
					Label:       orgUser.UserEmail,
					Value:       orgUser.UserId,
				})
			}
			orgUserSelection := cli.CreateSelector(cli.SelectorOpts{
				Choices: choices,
			})
			orgUserSelector := tea.NewProgram(orgUserSelection)
			if _, err := orgUserSelector.Run(); err != nil {
				return fmt.Errorf("failed to get user input: %w", err)
			}
			if orgUserSelection.GetExitCode() == cli.PromptCancelled {
				return errors.New("user cancelled")
			}
			userIdentifier = orgUserSelection.GetValue()
		}

		if _, err := uuid.Parse(userIdentifier); err != nil {
			if err := validate.Email(userIdentifier); err != nil {
				return fmt.Errorf("user identifier validation failed")
			}
		}

		if _, err := client.DeleteOrgUserV1(controller.DeleteOrgUserV1Input{
			OrgId:  org.Data.Id,
			UserId: userIdentifier,
		}); err != nil {
			switch true {
			case errors.Is(err, controller.ErrorInsufficientPermissions):
				{
					cli.PrintBoxedErrorMessage("You don't have sufficient permissions to remove a user")
					return fmt.Errorf("insufficient permissions")
				}
			case errors.Is(err, controller.ErrorOrgRequiresOneAdmin):
				{
					cli.PrintBoxedErrorMessage(
						"The user you are trying to remove is the last administrator which cannot be removed\n\n" +
							"Assign another user with the <admin> role before trying again",
					)
					return fmt.Errorf("last admin cannot be deleted")
				}
			}
			return fmt.Errorf("failed to delete user: %w", err)
		}

		fmt.Printf("✅ <%s> has been removed from your organisation\n", userIdEmailMap[userIdentifier])

		return nil
	},
}
