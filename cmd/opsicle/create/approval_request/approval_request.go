package approval_request

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/approvals"
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
	Use:     "approval-request <path-to-approval-request>",
	Aliases: []string{"appovreq", "appreq", "req", "ar"},
	Short:   "Creates an approval request given a path to an ApprovalRequest manifest",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		resourcePath, err := cli.GetFilePathFromArgs(args)
		if err != nil {
			return fmt.Errorf("failed to receive <path-to-approval-request>: %w", err)
		}
		approvalRequestInstance, err := approvals.LoadRequestFromFile(resourcePath)
		if err != nil {
			return fmt.Errorf("failed to load approval request: %w", err)
		}
		o, _ := json.MarshalIndent(approvalRequestInstance, "", "  ")
		logrus.Debugf("loaded approval request as follows:\n%s", string(o))

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
			Id:          fmt.Sprintf("%s/opsicle-create-approval-request", hostname),
		})
		if err != nil {
			return fmt.Errorf("failed to create client for approver service: %w", err)
		}
		requestUuid, err := client.CreateApprovalRequest(approverApi.CreateApprovalRequestInput{
			Callback:      approvalRequestInstance.Spec.Callback,
			Id:            approvalRequestInstance.Spec.Id,
			Links:         approvalRequestInstance.Spec.Links,
			Message:       approvalRequestInstance.Spec.Message,
			RequesterId:   approvalRequestInstance.Spec.RequesterId,
			RequesterName: approvalRequestInstance.Spec.RequesterName,
			Slack:         approvalRequestInstance.Spec.Slack,
			Telegram:      approvalRequestInstance.Spec.Telegram,
		})
		if err != nil {
			return fmt.Errorf("failed to create approval request: %w", err)
		}
		logrus.Infof("submitted request[%s]", requestUuid)
		return nil
	},
}
