package user

import (
	"errors"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/validate"
	"opsicle/pkg/controller"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "controller-url",
		Short:        'u',
		DefaultValue: "http://localhost:54321",
		Usage:        "Defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "user-id",
		Short:        'U',
		DefaultValue: "",
		Usage:        "ID (or email) of the user to add",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "template-id",
		Short:        't',
		DefaultValue: "",
		Usage:        "ID (or name) of the template to add a user to",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "can-invite",
		DefaultValue: false,
		Usage:        "When defined, the added user can invite/remove other users",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "can-delete",
		DefaultValue: false,
		Usage:        "When defined, the added user can delete this template",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "can-execute",
		DefaultValue: false,
		Usage:        "When defined, the added user can execute automations based on this template",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "can-update",
		DefaultValue: false,
		Usage:        "When defined, the added user can upload new versions of this template and update its details",
		Type:         cli.FlagTypeBool,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "user",
	Aliases: []string{"account", "acc", "u"},
	Short:   "Adds a user to an automation template from your library",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/list/templates"
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

		isUserIdValid := false
		isUserIdEmail := false
		userId := viper.GetString("user-id")
		if _, err := uuid.Parse(userId); err != nil { // not a uuid
			if err := validate.Email(userId); err == nil {
				isUserIdValid = true
				isUserIdEmail = true
			}
		} else {
			isUserIdValid = true
		}

		if !isUserIdValid {
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
						Placeholder: "User-to-add's email address",
						Type:        cli.PromptString,
					},
				},
				Title: "💬 What's the user's email address?",
			})
			userEmailPrompt := tea.NewProgram(userEmailInput)
			if _, err := userEmailPrompt.Run(); err != nil {
				return fmt.Errorf("failed to get user input: %w", err)
			}
			if userEmailInput.GetExitCode() == cli.PromptCancelled {
				fmt.Println("💬 Alrights, tell me again if you want to add a user")
				return errors.New("user cancelled action")
			}
			userId = userEmailInput.GetValue("email")

			if err := validate.Email(userId); err != nil {
				fmt.Printf("⚠️  The email '%s' doesn't look valid, try again:\n\n", userId)
				goto askForUserEmail
			}
			isUserIdEmail = true
		}

		templateId := viper.GetString("template-id")
		selectedTemplate, err := cli.HandleTemplateSelection(cli.HandleTemplateSelectionOpts{
			Client:             client,
			UserInput:          templateId,
			Prompt:             fmt.Sprintf("Which template should <%s> be added to?", userId),
			StartWithoutFilter: true,
		})
		if err != nil {
			if errors.Is(err, controller.ErrorNotFound) {
				cli.PrintBoxedInfoMessage(
					"You have no available templates to add users to -- submit a template first with `opsicle create template`",
				)
				return nil
			}
			return err
		}
		templateId = selectedTemplate.Id
		templateName := selectedTemplate.Name

		permissionsSelection := cli.CreateCheckboxes(cli.CreateCheckboxesOpts{
			Title: fmt.Sprintf(
				"What permissions should user %s have on template <%s>?",
				lipgloss.NewStyle().Bold(true).Render("<"+userId+">"),
				lipgloss.NewStyle().Bold(true).Render("<"+templateName+">"),
			),
			Items: []cli.CheckboxItem{
				{
					Id:          "can_view",
					Label:       "👀 Can View",
					Description: "User will be able to view details about this template",
				},
				{
					Id:          "can_update",
					Label:       "➕ Can Update",
					Description: "User will be able to update this template",
				},
				{
					Id:          "can_delete",
					Label:       "💥 Can Delete",
					Description: "User will be able to delete assets/versions in this template",
				},
				{
					Id:          "can_execute",
					Label:       "▶️  Can Execute",
					Description: "User will be able to execute this template",
				},
				{
					Id:          "can_invite",
					Label:       "👥 Can Invite",
					Description: "User will be able to invite/remove users from the template",
				},
			},
		})
		permissionsSelector := tea.NewProgram(permissionsSelection)
		checkboxModelOutput, err := permissionsSelector.Run()
		if err != nil {
			return fmt.Errorf("failed to execute selection of permissions: %w", err)
		}
		permissionsSelection = checkboxModelOutput.(*cli.CheckboxModel)
		if permissionsSelection.IsCancelled() {
			fmt.Println("💬 Alrights, tell me again if you want to add a user")
			return errors.New("user cancelled action")
		}

		fmt.Printf("⏳ Inviting user[%s] to template <%s>...\n", userId, templateName)

		addTemplateUserOpts := controller.CreateTemplateUserV1Input{
			TemplateId: templateId,
			CanView:    permissionsSelection.GetItemStatus("can_view"),
			CanExecute: permissionsSelection.GetItemStatus("can_execute"),
			CanInvite:  permissionsSelection.GetItemStatus("can_invite"),
			CanUpdate:  permissionsSelection.GetItemStatus("can_update"),
			CanDelete:  permissionsSelection.GetItemStatus("can_delete"),
		}
		if isUserIdEmail {
			addTemplateUserOpts.UserEmail = &userId
		} else if isUserIdValid {
			addTemplateUserOpts.UserId = &userId
		} else {
			cli.PrintBoxedErrorMessage(
				fmt.Sprintf("The provided user identifier <%s> doesn't look valid", userId),
			)
			return fmt.Errorf("invalid user id")
		}

		_, err = client.CreateTemplateUserV1(addTemplateUserOpts)
		if err != nil {
			return fmt.Errorf("failed to add user: %w", err)
		}

		cli.PrintBoxedSuccessMessage(
			fmt.Sprintf("If the user <%s> is on Opsicle, they've been invited to join you to work on template <%s>", userId, selectedTemplate.Name),
		)

		return nil
	},
}
