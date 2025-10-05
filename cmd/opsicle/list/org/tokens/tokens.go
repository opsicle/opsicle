package tokens

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"opsicle/internal/cli"
	"opsicle/pkg/controller"

	"github.com/olekukonko/tablewriter"
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
		Usage:        "codeword of the organisation to list tokens for",
		Type:         cli.FlagTypeString,
	},
}

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

		orgCode, err := cli.HandleOrgSelection(cli.HandleOrgSelectionOpts{
			Client:    client,
			UserInput: viper.GetString("org"),
		})
		if err != nil {
			return fmt.Errorf("org selection failed: %w", err)
		}
		if orgCode == nil {
			return errors.New("no organisation selected")
		}

		orgOutput, err := client.GetOrgV1(controller.GetOrgV1Input{Code: *orgCode})
		if err != nil {
			return fmt.Errorf("failed to retrieve org details: %w", err)
		}

		listOutput, err := client.ListOrgTokensV1(controller.ListOrgTokensV1Input{OrgId: orgOutput.Data.Id})
		if err != nil {
			switch {
			case errors.Is(err, controller.ErrorInsufficientPermissions):
				cli.PrintBoxedErrorMessage("You are not authorized to list tokens for this organization")
				return fmt.Errorf("not authorized to list tokens")
			case errors.Is(err, controller.ErrorNotFound):
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
			payload := struct {
				Org    controller.GetOrgV1OutputData        `json:"org"`
				Tokens controller.ListOrgTokensV1OutputData `json:"tokens"`
			}{
				Org:    orgOutput.Data,
				Tokens: listOutput.Data,
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(payload); err != nil {
				return fmt.Errorf("failed to encode json output: %w", err)
			}
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
					createdBy = token.CreatedBy.Id
				}
				lastUpdatedBy := ""
				if token.LastUpdatedBy != nil {
					lastUpdatedBy = token.LastUpdatedBy.Id
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
