package user

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/config"
	"opsicle/pkg/controller"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "org",
		DefaultValue: "",
		Usage:        "codeword of the organisation to check permissions in",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "user",
		DefaultValue: "",
		Usage:        "ID or email of the user to check permissions for",
		Type:         cli.FlagTypeString,
	},
}.Append(config.GetControllerUrlFlags())

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "user [action] [resourceType]",
	Short: "Checks if an organisation user can perform an action on a resource type",
	Args:  cobra.ExactArgs(2),
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		action := args[0]
		resource := args[1]
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/can/org/user"

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

		selectedOrg, err := cli.HandleOrgSelection(cli.HandleOrgSelectionOpts{
			Client:    client,
			UserInput: viper.GetString("org"),
		})
		if err != nil {
			return fmt.Errorf("org selection failed: %w", err)
		}

		org, err := client.GetOrgV1(controller.GetOrgV1Input{Ref: selectedOrg.Code})
		if err != nil {
			return fmt.Errorf("org retrieval failed: %w", err)
		}

		selectedUser, err := cli.HandleOrgUserSelection(cli.HandleOrgUserSelectionOpts{
			Client:    client,
			OrgId:     org.Data.Id,
			UserInput: viper.GetString("user"),
		})
		if err != nil {
			return fmt.Errorf("org user retrieval failed: %w", err)
		}

		canOutput, err := client.CanUserV1(controller.CanUserV1Input{
			OrgId:    org.Data.Id,
			UserId:   selectedUser.Id,
			Action:   action,
			Resource: resource,
		})
		if err != nil {
			return fmt.Errorf("permission check failed: %w", err)
		}

		var result controller.CanUserV1OutputData
		if canOutput != nil {
			result = canOutput.Data
		}
		if result.Action == "" {
			result.Action = action
		}
		if result.Resource == "" {
			result.Resource = resource
		}
		if result.OrgId == "" {
			result.OrgId = org.Data.Id
		}
		if result.UserId == "" {
			result.UserId = selectedUser.Id
		}

		switch viper.GetString("output") {
		case "json":
			display := struct {
				OrgId     string `json:"orgId"`
				OrgName   string `json:"orgName"`
				UserId    string `json:"userId"`
				UserEmail string `json:"userEmail"`
				Action    string `json:"action"`
				Resource  string `json:"resource"`
				Allows    uint64 `json:"allows"`
				Denys     uint64 `json:"denys"`
				IsAllowed bool   `json:"isAllowed"`
			}{
				OrgId:     result.OrgId,
				OrgName:   org.Data.Name,
				UserId:    selectedUser.Id,
				UserEmail: selectedUser.Email,
				Action:    result.Action,
				Resource:  result.Resource,
				Allows:    result.Allows,
				Denys:     result.Denys,
				IsAllowed: result.IsAllowed,
			}
			encoded, _ := json.MarshalIndent(display, "", "  ")
			fmt.Println(string(encoded))
		default:
			if result.IsAllowed {
				cli.PrintBoxedSuccessMessage(
					fmt.Sprintf(
						"User '%s' in org '%s' can %s %s",
						selectedUser.Email,
						selectedOrg.Code,
						result.Action,
						result.Resource,
					),
				)
			} else {
				cli.PrintBoxedErrorMessage(
					fmt.Sprintf(
						"User '%s' in org '%s' cannot %s %s",
						selectedUser.Email,
						selectedOrg.Code,
						result.Action,
						result.Resource,
					),
				)
			}
		}

		return nil
	},
}
