package audit_logs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"opsicle/internal/audit"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"
	"time"

	"github.com/olekukonko/tablewriter"
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
		Name:         "more",
		DefaultValue: false,
		Usage:        "When specified, also shows the source IP address and user agent",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "till",
		DefaultValue: time.Now().Local().Format(cli.TimestampSystem),
		Usage:        "Timestamp of the latest log entry to return, if --utc is not specified, this is in local time; Format it as {YYYY}-{MM}-{DD}T{HH}:{mm}:{ss}",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "utc",
		DefaultValue: false,
		Usage:        "When this is specified, all timestamps will be treated as UTC",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "limit",
		DefaultValue: 50,
		Usage:        "Defines how many audit log entries to return",
		Type:         cli.FlagTypeInteger,
	},
	{
		Name:         "reverse",
		DefaultValue: false,
		Usage:        "When this is specified, the latest logs will appear first",
		Type:         cli.FlagTypeBool,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "audit-logs",
	Aliases: []string{"al"},
	Short:   "Retrieves audit logs",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/list/audit-logs"
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

		cursor, err := time.Parse(cli.TimestampSystem, viper.GetString("till"))
		if err != nil {
			return fmt.Errorf("date format invalid")
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

		timezone := "UTC"
		if !viper.GetBool("utc") {
			timezone, _ = time.Now().Zone()
		}

		fmt.Printf("💭 Retrieving audit logs up till %s (%s)\n\n", cursor.Format("2 Jan 2006 3:04:05 PM"), timezone)
		auditLogsOutput, err := client.ListUserAuditLogsV1(controller.ListUserAuditLogsV1Input{
			Cursor:  cursor,
			Limit:   viper.GetInt64("limit"),
			Reverse: viper.GetBool("reverse"),
		})
		if err != nil {
			return fmt.Errorf("failed to list audit logs: %w", err)
		}
		switch viper.GetString("output") {
		case "json":
			o, _ := json.MarshalIndent(auditLogsOutput.Data, "", "  ")
			fmt.Println(string(o))
		case "text":
			fallthrough
		default:
			var tableData bytes.Buffer
			table := tablewriter.NewWriter(&tableData)
			timestampType := "local"
			if viper.GetBool("utc") {
				timestampType = "utc"
			}
			headers := []string{fmt.Sprintf("timestamp (%s)", timestampType)}
			if viper.GetBool("more") {
				headers = append(headers, "src ip", "src ua")
			}
			headers = append(headers, "description")
			table.Header(headers)
			for _, auditLog := range auditLogsOutput.Data {
				timestamp := auditLog.Timestamp
				if viper.GetBool("utc") {
					timestamp = timestamp.UTC()
				}
				row := []string{timestamp.Format(cli.TimestampSystem)}
				if viper.GetBool("more") {
					srcIp := "-"
					if auditLog.SrcIp != nil {
						srcIp = *auditLog.SrcIp
					}
					srcUa := "-"
					if auditLog.SrcUa != nil {
						srcUa = *auditLog.SrcUa
					}
					row = append(row, srcIp, srcUa)
				}
				row = append(row, audit.Interpret(auditLog))
				table.Append(row)
			}
			table.Render()
			fmt.Println(tableData.String())
		}
		return nil
	},
}
