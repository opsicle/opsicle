package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"opsicle/internal/cli"
	"opsicle/pkg/controller"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "controller-url",
		Short:        'u',
		DefaultValue: "http://localhost:54321",
		Usage:        "Defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "org",
		DefaultValue: "",
		Usage:        "Codeword of the organisation that the token should belong to",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "output-dir",
		DefaultValue: "",
		Usage:        "Location of the output directory (when specified, generated API key, certificate, and key will be placed in this directory)",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "token",
	Short: "Creates an access token for an organisation",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/create/org/token"
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

		orgCode, err := cli.HandleOrgSelection(cli.HandleOrgSelectionOpts{
			Client:    client,
			UserInput: viper.GetString("org"),
		})
		if err != nil {
			return fmt.Errorf("org selection failed: %w", err)
		}

		orgOutput, err := client.GetOrgV1(controller.GetOrgV1Input{Code: *orgCode})
		if err != nil {
			return fmt.Errorf("failed to retrieve org details: %w", err)
		}

		membershipOutput, err := client.GetOrgMembershipV1(controller.GetOrgMembershipV1Input{OrgId: orgOutput.Data.Id})
		if err != nil {
			return fmt.Errorf("failed to verify org membership: %w", err)
		}
		if !membershipOutput.Data.Permissions.CanManageUsers {
			cli.PrintBoxedErrorMessage("You are not authorized to manage tokens for this organization")
			return fmt.Errorf("not authorized to manage tokens")
		}

		formFields := cli.FormFields{
			{
				Id:          "token-name",
				Label:       "Token Name",
				Type:        cli.FormFieldString,
				IsRequired:  true,
				Description: "Used to identify the token later on. Only shown in the Opsicle UI",
			},
			{
				Id:          "token-description",
				Label:       "Token Description",
				Type:        cli.FormFieldString,
				IsRequired:  false,
				Description: "Optional context that explains where the token is used",
			},
		}
		tokenForm := cli.CreateForm(cli.FormOpts{
			Title:       fmt.Sprintf("Create token for %s", orgOutput.Data.Code),
			Description: "Provide a friendly name and optional description for the token",
			Fields:      formFields,
		})
		if err := tokenForm.GetInitWarnings(); err != nil {
			return fmt.Errorf("failed to initialize form: %w", err)
		}
		formProgram := tea.NewProgram(tokenForm)
		formOutput, err := formProgram.Run()
		if err != nil {
			return fmt.Errorf("failed to get user input: %w", err)
		}
		var ok bool
		tokenForm, ok = formOutput.(*cli.FormModel)
		if !ok {
			return fmt.Errorf("failed to receive a cli.FormModel")
		}
		if errors.Is(tokenForm.GetExitCode(), cli.ErrorUserCancelled) {
			return cli.ErrorUserCancelled
		}
		valueMap := tokenForm.GetValueMap()
		nameValue, _ := valueMap["token-name"].(string)
		nameValue = strings.TrimSpace(nameValue)
		descriptionValue, _ := valueMap["token-description"].(string)
		descriptionValue = strings.TrimSpace(descriptionValue)
		var descriptionPtr *string
		if descriptionValue != "" {
			desc := descriptionValue
			descriptionPtr = &desc
		}

		rolesOutput, err := client.ListOrgRolesV1(controller.ListOrgRolesV1Input{OrgId: orgOutput.Data.Id})
		if err != nil {
			return fmt.Errorf("failed to list org roles: %w", err)
		}
		if rolesOutput == nil || len(rolesOutput.Data) == 0 {
			return fmt.Errorf("no roles available for org %s", orgOutput.Data.Code)
		}

		sort.Slice(rolesOutput.Data, func(i, j int) bool {
			return strings.ToLower(rolesOutput.Data[i].Name) < strings.ToLower(rolesOutput.Data[j].Name)
		})
		fmt.Println("")
		fmt.Println("💬 Which role should this token assume?")
		fmt.Println("")

		selectorChoices := []cli.SelectorChoice{}
		for _, role := range rolesOutput.Data {
			description := []string{}
			for _, permission := range role.Permissions {
				description = append(description, fmt.Sprintf("%s", permission.Resource))
			}
			selectorChoices = append(selectorChoices, cli.SelectorChoice{
				Label:       role.Name,
				Description: strings.Join(description, ", "),
				Value:       role.Id,
			})
		}
		roleSelector := cli.CreateSelector(cli.SelectorOpts{Choices: selectorChoices})
		roleProgram := tea.NewProgram(roleSelector)
		if _, err := roleProgram.Run(); err != nil {
			return fmt.Errorf("failed to get role selection: %w", err)
		}
		if roleSelector.GetExitCode() == cli.PromptCancelled {
			return errors.New("user cancelled")
		}
		selectedRoleId := roleSelector.GetValue()

		createTokenOutput, err := client.CreateOrgTokenV1(controller.CreateOrgTokenV1Input{
			OrgId:       orgOutput.Data.Id,
			Name:        nameValue,
			Description: descriptionPtr,
			RoleId:      selectedRoleId,
		})
		if err != nil {
			cli.PrintBoxedErrorMessage(fmt.Sprintf("Failed to create token: %s", err))
			return fmt.Errorf("failed to create token")
		}
		if createTokenOutput == nil {
			return fmt.Errorf("controller returned no data")
		}

		logrus.Debugf("created token[%s] for org[%s]", createTokenOutput.Data.TokenId, orgOutput.Data.Id)

		outputDir := strings.TrimSpace(viper.GetString("output-dir"))
		if outputDir != "" {
			if err := os.MkdirAll(outputDir, 0o750); err != nil {
				return fmt.Errorf("failed to prepare output directory: %w", err)
			}
			tokenId := createTokenOutput.Data.TokenId
			outputTargets := writeTargets{
				{suffix: ".apikey", content: createTokenOutput.Data.ApiKey, description: "api key"},
				{suffix: ".crt", content: createTokenOutput.Data.CertificatePem, description: "certificate"},
				{suffix: ".key", content: createTokenOutput.Data.PrivateKeyPem, description: "private key"},
			}
			for _, target := range outputTargets {
				targetPath := filepath.Join(outputDir, fmt.Sprintf("%s%s", tokenId, target.suffix))
				if err := os.WriteFile(targetPath, []byte(target.content), 0o600); err != nil {
					return fmt.Errorf("failed to write %s to %s: %w", target.description, targetPath, err)
				}
				fmt.Printf("✨ Wrote %s data to %s\n", target.description, targetPath)
			}
		}

		outputFormat := strings.ToLower(viper.GetString("output"))
		if outputFormat == "json" {
			jsonOutput := struct {
				OrgCode string                                `json:"orgCode"`
				Token   controller.CreateOrgTokenV1OutputData `json:"token"`
			}{
				OrgCode: orgOutput.Data.Code,
				Token:   createTokenOutput.Data,
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(jsonOutput); err != nil {
				return fmt.Errorf("failed to encode json output: %w", err)
			}
			return nil
		}

		message := fmt.Sprintf("Created token '%s' for org '%s'", createTokenOutput.Data.Name, orgOutput.Data.Code)
		if outputDir != "" {
			message = fmt.Sprintf("%s\n\nThe API key, certificate, and private key can be found at: %s\n", message, outputDir)
		}
		cli.PrintBoxedSuccessMessage(message)
		if outputDir != "" {
			return nil
		}

		fmt.Printf("\n🔑 API Key:\n%s\n\n", createTokenOutput.Data.ApiKey)
		fmt.Println("📄 Certificate PEM:")
		fmt.Println(createTokenOutput.Data.CertificatePem)
		fmt.Println("🔐 Private Key PEM:")
		fmt.Println(createTokenOutput.Data.PrivateKeyPem)

		return nil
	},
}

type writeTargets []writeTarget
type writeTarget struct {
	suffix      string
	content     string
	description string
}
