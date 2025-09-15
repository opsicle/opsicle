package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"opsicle/internal/cli"
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
		Name:         "wide",
		Short:        'w',
		DefaultValue: false,
		Usage:        "Displays more information",
		Type:         cli.FlagTypeBool,
	},
	// {
	// 	Name:         "org",
	// 	DefaultValue: "",
	// 	Usage:        "Codeword of the organisation to list templates from",
	// 	Type:         cli.FlagTypeString,
	// },
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "templates",
	Aliases: []string{"tmpl", "t"},
	Short:   "Lists templates",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/list/templates"
		sessionToken, err := cli.RequireAuth(controllerUrl, methodId)
		if err != nil {
			fmt.Println("⚠️  You must be logged-in to run this command")
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

		templates, err := client.ListTemplatesV1(controller.ListTemplatesV1Input{
			Limit: 50,
		})
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}

		if len(templates.Data) == 0 {
			cli.PrintBoxedInfoMessage(
				"You don't have any templates, submit one using `opsicle submit template` and check back here",
			)
			return nil
		}

		switch viper.GetString("output") {
		case "json":
			o, _ := json.MarshalIndent(templates.Data, "", "  ")
			fmt.Println(string(o))
		case "text":

			var displayOut bytes.Buffer
			table := tablewriter.NewWriter(&displayOut)
			table.Configure(func(cfg *tablewriter.Config) {
				width, _, _ := term.GetSize(int(os.Stdout.Fd()))
				cfg.MaxWidth = width
			})
			var tableHeaders []any
			if viper.GetBool("wide") {
				tableHeaders = []any{"id", "name", "description", "version", "created by", "created at", "last updated by", "last updated at"}
			} else {
				tableHeaders = []any{"id", "name", "description", "version", "last updated at"}
			}
			table.Header(tableHeaders...)
			for _, template := range templates.Data {
				var tableRow []string
				if viper.GetBool("wide") {
					tableRow = []string{template.Id, template.Name, template.Description, strconv.Itoa(template.Version)}
					if template.CreatedBy != nil {
						tableRow = append(tableRow, template.CreatedBy.Email)
					} else {
						tableRow = append(tableRow, "-")
					}
					tableRow = append(tableRow, template.CreatedAt.Local().Format(cli.TimestampHuman))
					if template.LastUpdatedBy != nil {
						tableRow = append(tableRow, template.LastUpdatedBy.Email)
					} else {
						tableRow = append(tableRow, "-")
					}
					if template.LastUpdatedAt != nil {
						tableRow = append(tableRow, template.LastUpdatedAt.Local().Format(cli.TimestampHuman))
					} else {
						tableRow = append(tableRow, "-")
					}
				} else {
					tableRow = []string{template.Id, template.Name, template.Description, strconv.Itoa(template.Version)}
					if template.LastUpdatedAt != nil {
						tableRow = append(tableRow, template.LastUpdatedAt.Local().Format(cli.TimestampHuman))
					} else {
						tableRow = append(tableRow, "-")
					}
				}
				table.Append(tableRow)
			}
			table.Render()
			fmt.Println(displayOut.String())
		}

		return nil
	},
}
