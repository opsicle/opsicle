package org

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	{
		Name:         "org",
		DefaultValue: "",
		Usage:        "code of the orgnaisation to be created",
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
			return fmt.Errorf("failed to get a session token: %w", err)
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
			return fmt.Errorf("failed to create client for approver service: %w", err)
		}

		org, err := client.GetOrgV1(controller.GetOrgV1Input{
			Code: viper.GetString("org"),
		})
		if err != nil {
			if org.StatusCode == http.StatusUnauthorized {
				if err := controller.DeleteSessionToken(); err != nil {
					logrus.Warnf("failed to remove session token: %s", err)
				}
			}
			return fmt.Errorf("failed to retrieve current organisation: %w", err)
		}

		switch viper.GetString("output") {
		case "json":
			o, _ := json.MarshalIndent(org.Data, "", "  ")
			fmt.Println(string(o))
		case "text":
			fallthrough
		default:
			fmt.Printf("Id: %s\n", org.Data.Id)
			fmt.Printf("Name: %s\n", org.Data.Name)
			fmt.Printf("Code: %s\n", org.Data.Code)
			fmt.Printf("CreatedAt: %s\n", org.Data.CreatedAt.Local().Format("Jan 2 2006 03:04:05 PM"))
		}

		return nil
	},
}
