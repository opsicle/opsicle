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
	Use:     "approval-request",
	Aliases: []string{"approval-requests", "approvalrequests", "appovreq", "appreq", "reqs", "req", "ar"},
	Short:   "Retrieves ApprovalRequest keys from the approver service",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
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
			Id:          fmt.Sprintf("%s/opsicle-list-approval-request", hostname),
		})
		if err != nil {
			return fmt.Errorf("failed to create client for approver service: %s", err)
		}

		approvalRequestUuids, err := client.ListApprovalRequests()
		if err != nil {
			return fmt.Errorf("failed to retrieve approval requests: %s", err)
		}

		o, _ := json.MarshalIndent(approvalRequestUuids, "", "  ")
		fmt.Println(string(o))
		return nil
	},
}
