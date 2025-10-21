package org

import (
	"errors"
	"fmt"
	"net/http"
	"opsicle/cmd/opsicle/remove/org/user"
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
		Usage:        "defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "org",
		DefaultValue: "",
		Usage:        "code of the organisation to delete",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
	Command.AddCommand(user.Command)
}

var Command = &cobra.Command{
	Use:     "org",
	Aliases: []string{"organisation", "organization", "o"},
	Short:   "Deletes an organisation",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/remove/org"

	enforceAuth:
		sessionToken, err := cli.RequireAuth(controllerUrl, methodId)
		if err != nil {
			rootCmd := cmd.Root()
			rootCmd.SetArgs([]string{"login"})
			if _, err := rootCmd.ExecuteC(); err != nil {
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

		orgFlag := viper.GetString("org")
		var selectedOrg *cli.SelectedOrg
		orgSelectionOpts := cli.HandleOrgSelectionOpts{
			Client: client,
		}
		if orgFlag != "" {
			orgSelectionOpts.UserInput = orgFlag
		}
		selectedOrg, err = cli.HandleOrgSelection(orgSelectionOpts)
		if err != nil {
			return fmt.Errorf("org selection failed: %w", err)
		}

		orgDetails, err := client.GetOrgV1(controller.GetOrgV1Input{Ref: selectedOrg.Code})
		if err != nil {
			if orgDetails != nil && orgDetails.StatusCode == http.StatusUnauthorized {
				if err := controller.DeleteSessionToken(); err != nil {
					cli.PrintBoxedErrorMessage("Your session has expired and could not be refreshed automatically")
				}
				return fmt.Errorf("session token unauthorized")
			}
			return fmt.Errorf("failed to get organisation details: %w", err)
		}

		warningMessage := fmt.Sprintf(
			"You are about to permanently delete org <%s> (%s). This action cannot be undone. Continue?",
			orgDetails.Data.Name,
			orgDetails.Data.Code,
		)
		if err := cli.ShowConfirmation(cli.ShowConfirmationOpts{
			ColorAccent:     cli.AnsiRed,
			ColorForeground: cli.AnsiWhite,
			Title:           "🔴 Warning",
			Message:         warningMessage,
			ConfirmLabel:    "Delete",
			CancelLabel:     "Cancel",
		}); err != nil {
			switch {
			case errors.Is(err, cli.ErrorUserCancelled):
				fmt.Println("💬 Deletion cancelled.")
				return nil
			default:
				return fmt.Errorf("failed to show confirmation: %w", err)
			}
		}

		if _, err := client.DeleteOrgV1(controller.DeleteOrgV1Input{OrgId: orgDetails.Data.Id}); err != nil {
			switch {
			case errors.Is(err, controller.ErrorInsufficientPermissions):
				cli.PrintBoxedErrorMessage("You are not authorized to delete this organisation.")
				return fmt.Errorf("insufficient permissions")
			case errors.Is(err, controller.ErrorNotFound):
				cli.PrintBoxedErrorMessage("The organisation no longer exists.")
				return fmt.Errorf("organisation not found")
			default:
				return fmt.Errorf("failed to delete organisation: %w", err)
			}
		}

		fmt.Printf("✅ Organisation <%s> has been deleted.\n", orgDetails.Data.Code)
		return nil
	},
}
