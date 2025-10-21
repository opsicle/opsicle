package org

import (
	"errors"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/pkg/controller"

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
		Name:         "org",
		DefaultValue: "",
		Usage:        "Codeword for the organisation to add someone to",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "yes",
		DefaultValue: false,
		Usage:        "When this is specified, no warnings will be shown",
		Type:         cli.FlagTypeBool,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "org",
	Aliases: []string{"o"},
	Short:   "Leave an organisation",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)
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
		cli.PrintLogo()
		cli.PrintBoxedInfoMessage(
			"This command helps you to leave an organisation you're currently in",
		)
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

		if !viper.GetBool("yes") {
			if err := cli.ShowWarningWithConfirmation(
				"You are about to remove yourself from an org",
				true,
			); err != nil {
				return err
			}
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

		if !viper.GetBool("yes") {
			if err := cli.ShowWarningWithConfirmation(
				fmt.Sprintf(
					"You are about to remove yourself from <%s>.\n\nConfirm?",
					org.Data.Name,
				),
				true,
			); err != nil {
				return err
			}
		}

		if _, err := client.LeaveOrgV1(controller.LeaveOrgV1Input{
			OrgId: org.Data.Id,
		}); err != nil {
			if errors.Is(err, controller.ErrorOrgRequiresOneAdmin) {
				cli.PrintBoxedErrorMessage(
					"You are the last administrator of the organisation and cannot be removed\n\n" +
						"Assign another user with the <admin> role before trying again",
				)
				return fmt.Errorf("org requires at least one admin")
			}
			logrus.Errorf("failed to leave org: %s", err)
		}

		fmt.Printf("✅ You have left the organisation %s <%s>", org.Data.Name, org.Data.Code)

		return nil
	},
}
