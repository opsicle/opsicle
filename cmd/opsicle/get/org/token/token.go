package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"opsicle/internal/cli"
	"opsicle/pkg/controller"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
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
		Usage:        "codeword of the organisation to view tokens for",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "token [token-id]",
	Short: "Displays details for an organisation token",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/get/org/token"

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

		orgOutput, err := client.GetOrgV1(controller.GetOrgV1Input{Ref: *orgCode})
		if err != nil {
			return fmt.Errorf("failed to retrieve org details: %w", err)
		}

		membershipOutput, err := client.GetOrgMembershipV1(controller.GetOrgMembershipV1Input{OrgId: orgOutput.Data.Id})
		if err != nil {
			return fmt.Errorf("failed to verify org membership: %w", err)
		}
		if !membershipOutput.Data.Permissions.CanManageUsers {
			cli.PrintBoxedErrorMessage("You are not authorized to view tokens for this organization")
			return fmt.Errorf("not authorized to view tokens")
		}

		listOutput, err := client.ListOrgTokensV1(controller.ListOrgTokensV1Input{OrgId: orgOutput.Data.Id})
		if err != nil {
			switch {
			case errors.Is(err, controller.ErrorInsufficientPermissions):
				cli.PrintBoxedErrorMessage("You are not authorized to view tokens for this organization")
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
		if len(listOutput.Data) == 0 {
			fmt.Printf("No tokens found for org %s\n", orgOutput.Data.Code)
			return nil
		}

		tokens := listOutput.Data
		tokenMap := map[string]struct{}{}
		for _, token := range tokens {
			tokenMap[token.Id] = struct{}{}
		}

		userInputToken := ""
		if len(args) > 0 {
			userInputToken = args[0]
		}
		selectedTokenId := strings.TrimSpace(userInputToken)
		if selectedTokenId != "" {
			if _, err := uuid.Parse(selectedTokenId); err != nil {
				fmt.Printf("⚠️  The provided token id '%s' is not a valid UUID\n", selectedTokenId)
				selectedTokenId = ""
			} else if _, exists := tokenMap[selectedTokenId]; !exists {
				fmt.Printf("⚠️  The token '%s' does not belong to org %s\n", selectedTokenId, orgOutput.Data.Code)
				selectedTokenId = ""
			}
		}

		if selectedTokenId == "" {
			if len(tokens) == 1 {
				selectedTokenId = tokens[0].Id
			} else {
				fmt.Println("Select the token you would like to inspect:")
				fmt.Println("")
				choices := []cli.SelectorChoice{}
				for _, token := range tokens {
					description := ""
					if token.Description != nil {
						description = *token.Description
					}
					choices = append(choices, cli.SelectorChoice{
						Label:       fmt.Sprintf("%s (%s)", token.Name, token.Id),
						Description: description,
						Value:       token.Id,
					})
				}
				selector := cli.CreateSelector(cli.SelectorOpts{Choices: choices})
				program := tea.NewProgram(selector)
				if _, err := program.Run(); err != nil {
					return fmt.Errorf("failed to get user input: %w", err)
				}
				if selector.GetExitCode() == cli.PromptCancelled {
					return errors.New("user cancelled")
				}
				selectedTokenId = selector.GetValue()
			}
		}

		tokenOutput, err := client.GetOrgTokenV1(controller.GetOrgTokenV1Input{
			OrgId:   orgOutput.Data.Id,
			TokenId: selectedTokenId,
		})
		if err != nil {
			switch {
			case errors.Is(err, controller.ErrorInsufficientPermissions):
				cli.PrintBoxedErrorMessage("You are not authorized to view tokens for this organization")
				return fmt.Errorf("not authorized to view token")
			case errors.Is(err, controller.ErrorNotFound):
				cli.PrintBoxedErrorMessage("The token could not be found")
				return fmt.Errorf("token not found")
			default:
				return fmt.Errorf("failed to retrieve token details: %w", err)
			}
		}
		if tokenOutput == nil {
			return fmt.Errorf("controller returned no data")
		}

		outputFormat := strings.ToLower(viper.GetString("output"))
		switch outputFormat {
		case "json":
			payload := struct {
				Org   controller.GetOrgV1OutputData      `json:"org"`
				Token controller.GetOrgTokenV1OutputData `json:"token"`
			}{
				Org:   orgOutput.Data,
				Token: tokenOutput.Data,
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(payload); err != nil {
				return fmt.Errorf("failed to encode json output: %w", err)
			}
		default:
			description := ""
			if tokenOutput.Data.Description != nil {
				description = *tokenOutput.Data.Description
			}
			fmt.Println("")
			fmt.Printf("Organization: %s (%s)\n", orgOutput.Data.Code, orgOutput.Data.Name)
			fmt.Printf("Token ID: %s\n", tokenOutput.Data.Id)
			fmt.Printf("Name: %s\n", tokenOutput.Data.Name)
			fmt.Printf("Description: %s\n", fallbackString(description, "-"))
			fmt.Printf("Created At: %s\n", tokenOutput.Data.CreatedAt.Local().Format(cli.TimestampHuman))
			fmt.Printf("Last Updated At: %s\n", tokenOutput.Data.LastUpdatedAt.Local().Format(cli.TimestampHuman))
			fmt.Printf("Created By: %s\n", tokenOutput.Data.CreatedBy.Email)
			fmt.Printf("Last Updated By: %s\n", tokenOutput.Data.LastUpdatedBy.Email)

			if tokenOutput.Data.Role == nil {
				fmt.Println("Role: <none>")
				return nil
			}

			fmt.Printf("Role: %s (%s)\n", tokenOutput.Data.Role.Name, tokenOutput.Data.Role.Id)
			fmt.Println("")
			fmt.Println("Permissions:")
			table := cli.NewTable(cli.NewTableOpts{
				Headers: []string{"Permission ID", "Resource", "Allows", "Denys"},
				Rows: func(t *cli.Table) error {
					for idx, permission := range tokenOutput.Data.Role.Permissions {
						allows := formatActionMask(permission.Allows)
						denys := formatActionMask(permission.Denys)
						if err := t.NewRow(
							fallbackString(permission.Id, "-"),
							permission.Resource,
							allows,
							denys,
						); err != nil {
							return fmt.Errorf("failed to insert permission row[%d]: %w", idx, err)
						}
					}
					return nil
				},
			}).Render()
			fmt.Println(table.GetString())
		}

		return nil
	},
}

func formatActionMask(mask uint64) string {
	if mask == 0 {
		return "none"
	}
	actions := []string{}
	actionBits := []struct {
		bit  uint64
		name string
	}{
		{bit: 0b1, name: "create"},
		{bit: 0b10, name: "view"},
		{bit: 0b100, name: "update"},
		{bit: 0b1000, name: "delete"},
		{bit: 0b10000, name: "execute"},
		{bit: 0b100000, name: "manage"},
	}
	for _, action := range actionBits {
		if mask&action.bit != 0 {
			actions = append(actions, action.name)
		}
	}
	return strings.Join(actions, ", ")
}

func fallbackString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
