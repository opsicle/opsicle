package logout

import (
	"fmt"
	"net/http"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "controller-url",
		DefaultValue: "http://localhost:54321",
		Usage:        "defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "logout",
	Short: "Logs out of Opsicle from your terminal",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionToken, sessionFilePath, err := controller.GetSessionToken()
		if err != nil {
			fmt.Println("⚠️ You are not logged in, log in using `opsicle login`")
			return fmt.Errorf("you are not logged in")
		}

		controllerUrl := viper.GetString("controller-url")
		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			BearerAuth: &controller.NewClientBearerAuthOpts{
				Token: string(sessionToken),
			},
			Id: "opsicle/logout",
		})
		if err != nil {
			return fmt.Errorf("failed to create controller client: %s", err)
		}
		deleteSessionOutput, err := client.DeleteSessionV1()
		if err != nil {
			if deleteSessionOutput.StatusCode == http.StatusUnauthorized {
				fmt.Println("Existing session was invalid, please login again using `opsicle login")
			} else {
				logrus.Debugf("failed to delete session: %s", err)
				if err := controller.DeleteSessionToken(); err != nil {
					return fmt.Errorf("failed to remove file at path[%s], please do it yourself: %s", sessionFilePath, err)
				}
			}
		}
		if deleteSessionOutput.Data.IsSuccessful {
			if err := controller.DeleteSessionToken(); err != nil {
				return fmt.Errorf("failed to remove file at path[%s], please do it yourself: %s", sessionFilePath, err)
			}
			fmt.Printf("Session ID '%s' is now closed\n", deleteSessionOutput.Data.SessionId)
			fmt.Println("See you again <3")
		}
		return nil
	},
}
