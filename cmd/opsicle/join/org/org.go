package org

import (
	"errors"
	"fmt"
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
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "org",
	Aliases: []string{"j"},
	Short:   "Join an organisation",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/create/org/user"
		sessionToken, err := cli.RequireAuth(controllerUrl, methodId)
		if err != nil {
			fmt.Println("⚠️  You must be logged-in to run this command")
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

		listedOrgInvitations, err := client.ListOrgInvitationsV1()
		if err != nil {
			return fmt.Errorf("controller request failed: %w", err)
		}

		if len(listedOrgInvitations.Data.Invitations) > 0 {
			fmt.Printf("🔔 You have %v pending invitation(s) to organisations:\n\n", len(listedOrgInvitations.Data.Invitations))
		} else {
			fmt.Println("✅ You have no pending organisation invites")
			return nil
		}

		invitationIdJoinCodeMap := map[string]string{}
		selectorChoices := []cli.SelectorChoice{}
		for _, orgInvite := range listedOrgInvitations.Data.Invitations {
			invitationIdJoinCodeMap[orgInvite.Id] = orgInvite.JoinCode
			selectorChoices = append(selectorChoices, cli.SelectorChoice{
				Description: orgInvite.OrgName,
				Label:       orgInvite.OrgCode,
				Value:       orgInvite.Id,
			})
		}
		orgInvitationSelector := cli.CreateSelector(cli.SelectorOpts{
			Choices: selectorChoices,
		})
		orgInvitationProgram := tea.NewProgram(orgInvitationSelector)
		if _, err := orgInvitationProgram.Run(); err != nil {
			return fmt.Errorf("failed to get user input: %w", err)
		}
		if orgInvitationSelector.GetExitCode() == cli.PromptCancelled {
			return errors.New("user cancelled")
		}
		invitationId := orgInvitationSelector.GetValue()
		joinCode, ok := invitationIdJoinCodeMap[invitationId]
		if !ok {
			logrus.Errorf("unexpected error happened: selected invitationId wasn't matched with a join code")
			return fmt.Errorf("invalid selection")
		}
		acceptOrgInviteOutput, err := client.UpdateOrgInvitationV1(controller.UpdateOrgInvitationV1Input{
			Id:           invitationId,
			IsAcceptance: true,
			JoinCode:     joinCode,
		})
		if err != nil {
			return fmt.Errorf("failed operation: %w", err)
		}
		fmt.Printf(
			"✅ You are now a member of type[%s] in the org[%s]\n",
			acceptOrgInviteOutput.Data.MembershipType,
			acceptOrgInviteOutput.Data.OrgCode,
		)

		return nil
	},
}
