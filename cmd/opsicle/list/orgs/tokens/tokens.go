package tokens

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"opsicle/internal/cli"
	"opsicle/internal/config"
	"opsicle/internal/types"
	"opsicle/pkg/controller"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "org",
		DefaultValue: "",
		Usage:        "codeword of the organisation to list tokens for",
		Type:         cli.FlagTypeString,
	},
}.Append(config.GetControllerUrlFlags())

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "tokens",
	Short: "Lists tokens for an organisation",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/list/org/tokens"
	enforceAuth:
		sessionToken, err := cli.RequireAuth(controllerUrl, methodId)
		if err != nil {
			rootCmd := cmd.Root()
			rootCmd.SetArgs([]string{"login"})
			_, execErr := rootCmd.ExecuteC()
			if execErr != nil {
				return execErr
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

		orgOutput, err := client.GetOrgV1(controller.GetOrgV1Input{Ref: selectedOrg.Code})
		if err != nil {
			return fmt.Errorf("failed to retrieve org details: %w", err)
		}

		listOutput, err := client.ListOrgTokensV1(controller.ListOrgTokensV1Input{OrgId: orgOutput.Data.Id})
		if err != nil {
			switch {
			case errors.Is(err, types.ErrorInsufficientPermissions):
				cli.PrintBoxedErrorMessage("You are not authorized to list tokens for this organization")
				return fmt.Errorf("not authorized to list tokens")
			case errors.Is(err, types.ErrorNotFound):
				cli.PrintBoxedErrorMessage("The organization could not be found")
				return fmt.Errorf("organization not found")
			default:
				return fmt.Errorf("failed to list org tokens: %w", err)
			}
		}
		if listOutput == nil {
			return fmt.Errorf("controller returned no data")
		}

		outputFormat := strings.ToLower(viper.GetString("output"))
		switch outputFormat {
		case "json":
			o, _ := json.MarshalIndent(listOutput, "", "  ")
			fmt.Println(string(o))
		default:
			if len(listOutput.Data) == 0 {
				fmt.Printf("No tokens found for org %s\n", orgOutput.Data.Code)
				return nil
			}
			var buf bytes.Buffer
			table := tablewriter.NewWriter(&buf)
			table.Header([]any{"token id", "name", "description", "created at", "updated at", "created by", "updated by"}...)
			for _, token := range listOutput.Data {
				description := ""
				if token.Description != nil {
					description = *token.Description
				}
				createdBy := ""
				if token.CreatedBy != nil {
					createdBy = token.CreatedBy.Email
				}
				lastUpdatedBy := ""
				if token.LastUpdatedBy != nil {
					lastUpdatedBy = token.LastUpdatedBy.Email
				}
				table.Append([]string{
					token.Id,
					token.Name,
					description,
					token.CreatedAt.Local().Format(cli.TimestampHuman),
					token.LastUpdatedAt.Local().Format(cli.TimestampHuman),
					createdBy,
					lastUpdatedBy,
				})
			}
			table.Render()
			fmt.Println(buf.String())
		}

		return nil
	},
}
