package invitations

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "invitations",
	Aliases: []string{"invites", "i"},
	Short:   "Lists invites you have to organisations",
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

		listedOrgInvitations, err := client.ListOrgInvitationsV1()
		if err != nil {
			return fmt.Errorf("controller request failed: %w", err)
		}

		if len(listedOrgInvitations.Data.Invitations) > 0 {
			fmt.Printf("🔔 You have %v pending invitation(s) to organisations:\n\n", len(listedOrgInvitations.Data.Invitations))
		} else {
			fmt.Println("✅ You have no pending organisation invites")
			return nil
		}

		outputFormat := viper.GetString("output")
		switch outputFormat {
		case "json":
			o, _ := json.MarshalIndent(listedOrgInvitations.Data.Invitations, "", "  ")
			fmt.Println(string(o))
		case "text":
			fallthrough
		default:
			var displayOut bytes.Buffer
			table := tablewriter.NewWriter(&displayOut)
			table.Header([]any{"org", "code", "inviter", "invited at"}...)
			for _, invitation := range listedOrgInvitations.Data.Invitations {
				table.Append([]string{invitation.OrgName, invitation.OrgCode, invitation.InviterEmail, invitation.InvitedAt.Local().Format("Jan 2 2006 03:04:05 PM")})
			}
			table.Render()
			fmt.Println(displayOut.String())
		}

		if len(listedOrgInvitations.Data.Invitations) > 0 {
			fmt.Printf("ℹ️  To join an organisation, use `opsicle join org`\n")
		}

		return nil
	},
}
