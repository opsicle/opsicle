package template

import (
	"errors"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	Use:     "template",
	Aliases: []string{"t"},
	Short:   "Accept invitations to collaborate on automation templates",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/join/template"
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

		listUserTemplateInvitationsV1Output, err := client.ListUserTemplateInvitationsV1()
		if err != nil {
			return fmt.Errorf("controller request failed: %w", err)
		}
		invitations := listUserTemplateInvitationsV1Output.Data.Invitations

		if len(invitations) > 0 {
			fmt.Printf("🔔 You have %v pending invitation(s) to collaborate on automation tempaltes:\n\n", len(invitations))
		} else {
			fmt.Println("✅ You have no pending invites to automation templates")
			return nil
		}

		invitationIdJoinCodeMap := map[string]controller.ListUserTemplateInvitationsV1OutputDataInvitation{}
		selectorChoices := []cli.SelectorChoice{}
		for _, invitation := range invitations {
			invitationIdJoinCodeMap[invitation.Id] = invitation
			selectorChoices = append(selectorChoices, cli.SelectorChoice{
				Description: fmt.Sprintf("Invited by %s at %s", invitation.InviterEmail, invitation.InvitedAt.Local().Format(cli.TimestampHuman)),
				Label:       invitation.TemplateName,
				Value:       invitation.Id,
			})
		}
		templateInvitationSelector := cli.CreateSelector(cli.SelectorOpts{
			Choices: selectorChoices,
		})
		templateInvitationProgram := tea.NewProgram(templateInvitationSelector)
		if _, err := templateInvitationProgram.Run(); err != nil {
			return fmt.Errorf("failed to get user input: %w", err)
		}
		if templateInvitationSelector.GetExitCode() == cli.PromptCancelled {
			return errors.New("user cancelled")
		}
		invitationId := templateInvitationSelector.GetValue()
		invitation, ok := invitationIdJoinCodeMap[invitationId]
		if !ok {
			logrus.Errorf("unexpected error happened: selected invitationId wasn't matched with a join code")
			return fmt.Errorf("invalid selection")
		}

		isUserAccept := true
		if err := cli.ShowConfirmation(cli.ShowConfirmationOpts{
			Title:       "🟡 Accept invitation?",
			ColorAccent: cli.AnsiYellow,
			Message: fmt.Sprintf(
				lipgloss.NewStyle().Italic(true).Render("💡 Verify that this invitation is legitimate before hitting accept, malicious users may invite you to convince you to execute unwanted code on your infrastructure")+
					"\n\n"+
					"You are about to accept an invitation to collaborate on template <%s> by <%s>, are you sure?",
				invitation.TemplateName,
				invitation.InviterEmail,
			),
			ConfirmLabel: "Accept",
			CancelLabel:  "Decline",
		}); err != nil {
			isUserAccept = false
			if !errors.Is(err, cli.ErrorUserCancelled) {
				return fmt.Errorf("unexpected error: %w", err)
			}
		}

		updateTemplateInvitationOutput, err := client.UpdateTemplateInvitationV1(controller.UpdateTemplateInvitationV1Input{
			Id:           invitationId,
			IsAcceptance: isUserAccept,
			JoinCode:     invitation.JoinCode,
		})
		if err != nil {
			return fmt.Errorf("failed to update template invitation: %w", err)
		}
		if updateTemplateInvitationOutput.Data.IsSuccessful {
			if updateTemplateInvitationOutput.Data.TemplateUser != nil {
				templateUser := updateTemplateInvitationOutput.Data.TemplateUser
				permissions := []string{}
				permissions = append(permissions, fmt.Sprintf("Can view: %v", templateUser.CanView))
				permissions = append(permissions, fmt.Sprintf("Can delete template: %v", templateUser.CanDelete))
				permissions = append(permissions, fmt.Sprintf("Can execute template: %v", templateUser.CanExecute))
				permissions = append(permissions, fmt.Sprintf("Can update template: %v", templateUser.CanUpdate))
				permissions = append(permissions, fmt.Sprintf("Can invite other users: %v", templateUser.CanInvite))
				cli.PrintBoxedSuccessMessage(
					fmt.Sprintf(
						"You have accepted the invitation from <%s> and are now a collaborator on template <%s> with ID '%s'. Your permissions are as follows:\n\n%s\n\nUse `opsicle list templates` to see templates you have access to",
						invitation.InviterEmail,
						invitation.TemplateName,
						templateUser.TemplateId,
						strings.Join(permissions, "\n"),
					),
				)
			} else {
				cli.PrintBoxedInfoMessage(
					fmt.Sprintf(
						"You have rejected the invitation from <%s> to collaborate on template <%s>. If this was spam, please report it to us at abuse@opsicle.io",
						invitation.InviterEmail,
						invitation.TemplateName,
					),
				)
			}
		}

		return nil
	},
}
