package org

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"
	"os"

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
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "org",
	Aliases: []string{"o"},
	Short:   "Retrieves information about your current organisation",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionToken, sessionFilePath, err := controller.GetSessionToken()
		if err != nil {
			return fmt.Errorf("failed to get a session token: %s", err)
		}
		logrus.Debugf("loaded credentials from path[%s]", sessionFilePath)

		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown-host"
		}
		controllerUrl := viper.GetString("controller-url")
		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			BearerAuth: &controller.NewClientBearerAuthOpts{
				Token: sessionToken,
			},
			Id: fmt.Sprintf("%s/opsicle-get-org", hostname),
		})
		if err != nil {
			return fmt.Errorf("failed to create client for approver service: %s", err)
		}

		org, _, err := client.GetOrgV1()
		if err != nil {
			return fmt.Errorf("failed to retrieve current organisation: %s", err)
		}

		o, _ := json.MarshalIndent(org, "", "  ")
		fmt.Println(string(o))
		return nil
	},
}
