package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	"opsicle/internal/cli"
	"opsicle/internal/config"
	"opsicle/pkg/controller"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{}.Append(config.GetControllerUrlFlags())

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "users",
	Aliases: []string{"u"},
	Short:   "Retrieves users from the controller service",
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
			Id: fmt.Sprintf("%s/opsicle-get-users", hostname),
		})
		if err != nil {
			return fmt.Errorf("failed to create client for approver service: %w", err)
		}

		listUsersOutput, err := client.ListUsersV1()
		if err != nil {
			if listUsersOutput.Response.StatusCode == http.StatusUnauthorized {
				if err := controller.DeleteSessionToken(); err != nil {
					logrus.Warnf("failed to remove session token: %s", err)
				}
			}
			return fmt.Errorf("failed to retrieve users: %w", err)
		}

		o, _ := json.MarshalIndent(listUsersOutput.Data, "", "  ")
		fmt.Println(string(o))
		return nil
	},
}
