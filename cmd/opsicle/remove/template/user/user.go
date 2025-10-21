package user

import (
	"errors"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"

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
		Usage:        "ID (or email) of the user to remove",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "template-id",
		Short:        't',
		DefaultValue: "",
		Usage:        "ID (or name) of the template to remove a user from",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "user",
	Aliases: []string{"account", "acc", "u"},
	Short:   "Removes a user from a defined automation templates from your library",
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

		inputTemplateName := viper.GetString("template-id")
		templateInstance, err := cli.HandleTemplateSelection(cli.HandleTemplateSelectionOpts{
			Client:    client,
			UserInput: inputTemplateName,
			// ServiceLog: servicesLogs,
		})
		if err != nil {
			return fmt.Errorf("failed to select a template: %w", err)
		}

		templateUsers, err := client.ListTemplateUsersV1(controller.ListTemplateUsersV1Input{
			TemplateId: templateInstance.Id,
		})
		if err != nil {
			return fmt.Errorf("failed to get template users: %w", err)
		}
		if len(templateUsers.Data.Users) == 1 {
			cli.PrintBoxedErrorMessage(
				fmt.Sprintf("You are the last user associated with template <%s> and cannot be removed", templateInstance.Name),
			)
			return fmt.Errorf("someone's got to own this man")
		}

		selectedUser, err := cli.HandleTemplateUserSelection(
			cli.HandleTemplateUserSelectionOpts{
				Client:     client,
				UserInput:  viper.GetString("user-id"),
				TemplateId: templateInstance.Id,
				// ServiceLog: servicesLogs,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to select template user: %w", err)
		}

		warningMessage := fmt.Sprintf(
			"You are about to remove <%s>'s access to template <%s>",
			selectedUser.Email,
			templateInstance.Name,
		)
		warningMessage += "\n\nHit OK below to confirm"
		if err := cli.ShowWarningWithConfirmation(
			warningMessage,
			false,
		); err != nil {
			return err
		}

		fmt.Printf("⏳ Removing user[%s] with id[%s] from template[%s]...\n", selectedUser.Email, selectedUser.Id, templateInstance.Id)

		deleteUserOutput, err := client.DeleteTemplateUserV1(controller.DeleteTemplateUserV1Input{
			TemplateId: templateInstance.Id,
			UserId:     selectedUser.Id,
		})
		if err != nil {
			if errors.Is(err, controller.ErrorLastUserInResource) {
				cli.PrintBoxedErrorMessage(
					fmt.Sprintf("<%s> is the last user associated with template <%s> and cannot be removed", selectedUser.Email, templateInstance.Name),
				)
				return err
			} else if errors.Is(err, controller.ErrorLastManagerOfResource) {
				cli.PrintBoxedErrorMessage(
					fmt.Sprintf("<%s> is the last manager of template <%s> and cannot be removed", selectedUser.Email, templateInstance.Name),
				)
				return err
			} else if errors.Is(err, controller.ErrorInsufficientPermissions) {
				cli.PrintBoxedErrorMessage(
					fmt.Sprintf("You do not have sufficient permissions to remove a user from template <%s>", templateInstance.Name),
				)
				return err
			} else if errors.Is(err, controller.ErrorNotFound) {
				cli.PrintBoxedErrorMessage(
					fmt.Sprintf("User <%s> doesn't seem to have access to template <%s>", selectedUser.Email, templateInstance.Name),
				)
				return err
			}
			return fmt.Errorf("failed to delete template user: %w", err)
		}

		if deleteUserOutput.Data.IsSuccessful {
			cli.PrintBoxedSuccessMessage(
				fmt.Sprintf("Removed user <%s> from template <%s>", selectedUser.Email, templateInstance.Id),
			)
		}

		return nil
	},
}
