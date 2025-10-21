package roles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"opsicle/internal/cli"
	"opsicle/pkg/controller"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	actionCreateMask uint64 = 1 << iota
	actionViewMask
	actionUpdateMask
	actionDeleteMask
	actionExecuteMask
	actionManageMask
)

var actionMaskLabels = []struct {
	Mask  uint64
	Label string
}{
	{Mask: actionCreateMask, Label: "create"},
	{Mask: actionViewMask, Label: "view"},
	{Mask: actionUpdateMask, Label: "update"},
	{Mask: actionDeleteMask, Label: "delete"},
	{Mask: actionExecuteMask, Label: "execute"},
	{Mask: actionManageMask, Label: "manage"},
}

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
		Usage:        "codeword of the organisation to list roles from",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "roles",
	Aliases: []string{"role", "r"},
	Short:   "Lists roles in an organisation you are in",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/list/org/roles"
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

		roles, err := client.ListOrgRolesV1(controller.ListOrgRolesV1Input{OrgId: org.Data.Id})
		if err != nil {
			return fmt.Errorf("org role retrieval failed: %w", err)
		}
		if roles == nil {
			return fmt.Errorf("failed to retrieve org roles")
		}

		switch strings.ToLower(viper.GetString("output")) {
		case "json":
			payload, marshalErr := json.MarshalIndent(roles.Data, "", "  ")
			if marshalErr != nil {
				return fmt.Errorf("failed to marshal output: %w", marshalErr)
			}
			fmt.Println(string(payload))
			return nil
		case "text":
			fallthrough
		default:
			var displayOut bytes.Buffer
			table := tablewriter.NewWriter(&displayOut)
			table.Header([]any{"name", "created", "updated", "created by", "permissions"}...)
			for _, role := range roles.Data {
				createdAt := renderTimestamp(role.CreatedAt)
				updatedAt := renderTimestamp(role.LastUpdatedAt)
				createdBy := renderCreatedBy(role.CreatedBy)
				permissions := renderPermissions(role.Permissions)
				table.Append([]string{role.Name, createdAt, updatedAt, createdBy, permissions})
			}
			table.Render()
			fmt.Println(displayOut.String())
		}
		return nil
	},
}

func renderTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.Local().Format(cli.TimestampHuman)
}

func renderCreatedBy(user *controller.ListOrgRolesV1OutputDataRoleUser) string {
	if user == nil {
		return "-"
	}
	if user.Email != "" {
		return user.Email
	}
	if user.Id != "" {
		return user.Id
	}
	return "-"
}

func renderPermissions(perms []controller.ListOrgRolesV1OutputDataRolePermission) string {
	if len(perms) == 0 {
		return "-"
	}
	descriptions := make([]string, 0, len(perms))
	for _, perm := range perms {
		allow := formatActions(perm.Allows)
		deny := formatActions(perm.Denys)
		if deny != "-" {
			descriptions = append(descriptions, fmt.Sprintf("%s allow=[%s] deny=[%s]", perm.Resource, allow, deny))
			continue
		}
		descriptions = append(descriptions, fmt.Sprintf("%s allow=[%s]", perm.Resource, allow))
	}
	return strings.Join(descriptions, "\n")
}

func formatActions(mask uint64) string {
	if mask == 0 {
		return "-"
	}
	labels := make([]string, 0, len(actionMaskLabels))
	for _, action := range actionMaskLabels {
		if mask&action.Mask != 0 {
			labels = append(labels, action.Label)
		}
	}
	if len(labels) == 0 {
		return "-"
	}
	return strings.Join(labels, ", ")
}
