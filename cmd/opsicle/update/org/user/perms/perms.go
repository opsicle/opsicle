package perms

import (
	"errors"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/validate"
	"opsicle/pkg/controller"
	"sort"

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
	{
		Name:         "member-type",
		DefaultValue: "",
		Usage:        "Type of membership that the user should be updated to",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "perms",
	Aliases: []string{"p"},
	Short:   "Updates organisation user permissions",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/update/org/user/perms"
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
			Client:    client,
			UserInput: viper.GetString("org"),
		})
		if err != nil {
			return fmt.Errorf("org selection failed: %w", err)
		}
		org, err := client.GetOrgV1(controller.GetOrgV1Input{Ref: selectedOrg.Code})
		if err != nil {
			return fmt.Errorf("org retrieval failed: %w", err)
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
			if _, ok := userIdEmailMap[userIdentifier]; ok {
				isUserIdentifiedValid = true
			}
		}
		if !isUserIdentifiedValid {
			if isUserIdentifiedSpecified {
				fmt.Printf("⚠️  It looks like your specified user '%s' does not exist in the organisation\n", userIdentifier)
				fmt.Println("💬 Please select a user from your organisation:")
			} else {
				fmt.Println("💬 Select the user from your organisation that you want to change permissions for:")
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

		updateOrgUserOpts := controller.UpdateOrgUserV1Input{
			OrgId: org.Data.Id,
			User:  userIdentifier,
			Update: map[string]any{
				"type": membershipType,
			},
		}
		updateOrgUserOutput, err := client.UpdateOrgUserV1(updateOrgUserOpts)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
		if updateOrgUserOutput.Data.IsSuccessful {
			fmt.Printf("✅ The user <%s> is now of type '%s' in your organisation\n", userIdEmailMap[userIdentifier], membershipType)
		}

		return nil
	},
}
