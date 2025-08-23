package approval_request

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	approverApi "opsicle/pkg/approver"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "approver-url",
		Short:        'u',
		DefaultValue: "http://localhost:12345",
		Usage:        "defines the url where the approver service is accessible at",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "approval-request <request-uuid>",
	Aliases: []string{"appovreq", "appreq", "req", "ar"},
	Short:   "Retrieves an ApprovalRequest given the request UUID",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("failed to receive <request-uuid")
		}
		requestUuid := args[0]

		approverUrl := viper.GetString("approver-url")
		logrus.Infof("using approver service at url[%s]", approverUrl)

		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)

		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown-host"
		}
		client, err := approverApi.NewClient(approverApi.NewClientOpts{
			ApproverUrl: approverUrl,
			Id:          fmt.Sprintf("%s/opsicle-get-approval-request", hostname),
		})
		if err != nil {
			return fmt.Errorf("failed to create client for approver service: %w", err)
		}

		approvalRequest, err := client.GetApprovalRequest(requestUuid)
		if err != nil {
			return fmt.Errorf("failed to retrieve approval request status of request[%s]: %s", requestUuid, err)
		}

		o, _ := json.MarshalIndent(approvalRequest, "", "  ")
		fmt.Println(string(o))
		return nil
	},
}
