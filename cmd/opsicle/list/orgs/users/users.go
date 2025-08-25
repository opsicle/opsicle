package users

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/olekukonko/tablewriter"
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
		Usage:        "Codeword of the organisation to list users from",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "users",
	Aliases: []string{"user", "u"},
	Short:   "Lists users in an organisation you are in",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/list/org/invitations"
		sessionToken, err := cli.RequireAuth(controllerUrl, methodId)
		if err != nil {
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

		orgCode := viper.GetString("org")
		if orgCode == "" {
			logrus.Debugf("retrieving available organisations to current user")
			listOrgsOutput, err := client.ListOrgsV1()
			if err != nil {
				return fmt.Errorf("controller request failed")
			}

			fmt.Println("Select an organisation to list users from:")
			fmt.Println("")
			choices := []cli.SelectorChoice{}
			for _, org := range listOrgsOutput.Data {
				choices = append(choices, cli.SelectorChoice{
					Description: org.Name,
					Label:       org.Code,
					Value:       org.Code,
				})
			}
			orgSelection := cli.CreateSelector(cli.SelectorOpts{
				Choices: choices,
			})
			orgSelector := tea.NewProgram(orgSelection)
			if _, err := orgSelector.Run(); err != nil {
				return fmt.Errorf("failed to get user input: %w", err)
			}
			if orgSelection.GetExitCode() == cli.PromptCancelled {
				return errors.New("user cancelled")
			}
			orgCode = orgSelection.GetValue()
		}
		org, err := client.GetOrgV1(controller.GetOrgV1Input{
			Code: orgCode,
		})
		if err != nil {
			return fmt.Errorf("org retrieval failed: %w", err)
		}
		orgUsers, err := client.ListOrgUsersV1(controller.ListOrgUsersV1Input{
			OrgId: org.Data.Id,
		})
		if err != nil {
			return fmt.Errorf("org user retrieval failed: %w", err)
		}

		switch viper.GetString("output") {
		case "json":
			o, _ := json.MarshalIndent(orgUsers.Data, "", "  ")
			fmt.Println(string(o))
		case "text":

			var displayOut bytes.Buffer
			table := tablewriter.NewWriter(&displayOut)
			table.Header([]any{"email", "member type", "joined"}...)
			for _, orgUser := range orgUsers.Data {
				table.Append([]string{orgUser.UserEmail, orgUser.MemberType, orgUser.JoinedAt.Local().Format("Jan 2 2006 03:04:05 PM")})
			}
			table.Render()
			fmt.Println(displayOut.String())
		}

		return nil
	},
}
