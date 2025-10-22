package templates

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/types"
	"opsicle/pkg/controller"
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
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
		Usage:        "Codeword of the organisation to list templates for",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "wide",
		Short:        'w',
		DefaultValue: false,
		Usage:        "Displays more information",
		Type:         cli.FlagTypeBool,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "templates",
	Aliases: []string{"template"},
	Short:   "Lists templates for an organisation",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/list/org/templates"
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

		listOutput, err := client.ListOrgTemplatesV1(controller.ListOrgTemplatesV1Input{
			OrgId: orgOutput.Data.Id,
			Limit: 50,
		})
		if err != nil {
			switch {
			case errors.Is(err, types.ErrorInsufficientPermissions):
				cli.PrintBoxedErrorMessage("You are not authorized to list templates for this organization")
				return fmt.Errorf("not authorized to list org templates")
			case errors.Is(err, types.ErrorNotFound):
				cli.PrintBoxedErrorMessage("The organization could not be found")
				return fmt.Errorf("organization not found")
			default:
				return fmt.Errorf("failed to list org templates: %w", err)
			}
		}
		if listOutput == nil {
			return fmt.Errorf("controller returned no data")
		}

		if len(listOutput.Data) == 0 {
			cli.PrintBoxedInfoMessage(fmt.Sprintf("Organization %s does not have any templates yet", orgOutput.Data.Code))
			return nil
		}

		switch viper.GetString("output") {
		case "json":
			o, _ := json.MarshalIndent(listOutput.Data, "", "  ")
			fmt.Println(string(o))
		default:
			var displayOut bytes.Buffer
			table := tablewriter.NewWriter(&displayOut)
			table.Configure(func(cfg *tablewriter.Config) {
				width, _, _ := term.GetSize(int(os.Stdout.Fd()))
				cfg.MaxWidth = width
			})
			var headers []any
			if viper.GetBool("wide") {
				headers = []any{"id", "name", "description", "version", "created by", "created at", "last updated by", "last updated at"}
			} else {
				headers = []any{"id", "name", "description", "version", "last updated at"}
			}
			table.Header(headers...)
			for _, template := range listOutput.Data {
				var row []string
				version := strconv.Itoa(template.Version)
				lastUpdatedAt := "-"
				if template.LastUpdatedAt != nil {
					lastUpdatedAt = template.LastUpdatedAt.Local().Format(cli.TimestampHuman)
				}
				if viper.GetBool("wide") {
					createdBy := "-"
					if template.CreatedBy != nil {
						createdBy = template.CreatedBy.Email
					}
					createdAt := template.CreatedAt.Local().Format(cli.TimestampHuman)
					lastUpdatedBy := "-"
					if template.LastUpdatedBy != nil {
						lastUpdatedBy = template.LastUpdatedBy.Email
					}
					row = []string{
						template.Id,
						template.Name,
						template.Description,
						version,
						createdBy,
						createdAt,
						lastUpdatedBy,
						lastUpdatedAt,
					}
				} else {
					row = []string{
						template.Id,
						template.Name,
						template.Description,
						version,
						lastUpdatedAt,
					}
				}
				table.Append(row)
			}
			table.Render()
			fmt.Println(displayOut.String())
		}

		return nil
	},
}
