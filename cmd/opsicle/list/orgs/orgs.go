package orgs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"opsicle/cmd/opsicle/list/orgs/invitations"
	"opsicle/cmd/opsicle/list/orgs/users"
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
}

func init() {
	Command.AddCommand(invitations.Command)
	Command.AddCommand(users.Command)
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "orgs",
	Aliases: []string{"organisations", "organizations", "org", "o"},
	Short:   "Lists your organisations in your Opsicle instance",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionToken, _, err := controller.GetSessionToken()
		if err != nil {
			fmt.Println("⚠️ You must be logged-in to run this command")
			return fmt.Errorf("login is required")
		}

		controllerUrl := viper.GetString("controller-url")
		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			BearerAuth: &controller.NewClientBearerAuthOpts{
				Token: sessionToken,
			},
			Id: "opsicle/list/orgs",
		})
		if err != nil {
			return fmt.Errorf("failed to create controller client: %w", err)
		}

		output, err := client.ValidateSessionV1()
		if err != nil || output.Data.IsExpired {
			if err := controller.DeleteSessionToken(); err != nil {
				fmt.Printf("⚠️ We failed to remove the session token for you, please do it yourself\n")
			}
			fmt.Println("⚠️  Please login again using `opsicle login`")
			return fmt.Errorf("session invalid")
		}

		listOrgsOutput, err := client.ListOrgsV1()
		if err != nil {
			return fmt.Errorf("controller request failed")
		}

		outputFormat := viper.GetString("output")
		switch outputFormat {
		case "json":
			o, _ := json.MarshalIndent(listOrgsOutput.Data, "", "  ")
			fmt.Println(string(o))
		case "text":
			fallthrough
		default:
			var displayOut bytes.Buffer
			table := tablewriter.NewWriter(&displayOut)
			table.Header([]any{"code", "member type", "joined at"}...)
			for _, org := range listOrgsOutput.Data {
				table.Append([]string{org.Code, org.MemberType, org.JoinedAt.Local().Format(cli.TimestampHuman)})
			}
			table.Render()
			fmt.Println(displayOut.String())
		}

		return nil
	},
}
