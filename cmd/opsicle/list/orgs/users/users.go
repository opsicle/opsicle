package users

import (
	"bytes"
	"encoding/json"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/config"
	"opsicle/pkg/controller"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "org",
		DefaultValue: "",
		Usage:        "Codeword of the organisation to list users from",
		Type:         cli.FlagTypeString,
	},
}.Append(config.GetControllerUrlFlags())

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
			table.Header([]any{"email", "member type", "roles", "joined"}...)
			for _, orgUser := range orgUsers.Data {
				roleNames := make([]string, 0, len(orgUser.Roles))
				for _, role := range orgUser.Roles {
					roleNames = append(roleNames, role.Name)
				}
				table.Append([]string{orgUser.UserEmail, orgUser.MemberType, strings.Join(roleNames, ", "), orgUser.JoinedAt.Local().Format(cli.TimestampHuman)})
			}
			table.Render()
			fmt.Println(displayOut.String())
		}

		return nil
	},
}
