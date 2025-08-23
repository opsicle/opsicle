package approval

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/approvals"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	approverApi "opsicle/pkg/approver"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "approver-url",
		DefaultValue: "http://localhost:12345",
		Usage:        "defines the url where the approver service is accessible at",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "retry-interval",
		DefaultValue: 5 * time.Second,
		Usage:        "defines the retry interval for retrieving the status",
		Type:         cli.FlagTypeDuration,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "approval <path-to-approval-request>",
	Short: "Runs an approval manifest given a path to an ApprovalRequest manifest",
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

		client, err := approverApi.NewClient(approverApi.NewClientOpts{
			ApproverUrl: approverUrl,
			Id:          "opsicle-run-approval",
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
		retryInterval := viper.GetDuration("retry-interval")
		logrus.Infof("checks will be done at %v intervals, set log level to debug to see intervals if needed", retryInterval)

		for {
			logrus.Infof("checking status of request[%s]...", requestUuid)
			approvalRequest, err := client.GetApprovalRequest(requestUuid)
			if err != nil {
				logrus.Errorf("failed to retrieve approval request status of request[%s]: %s", requestUuid, err)
				continue
			}
			if approvalRequest.Approval == nil {
				logrus.Debugf("approval not received, waiting for %v before trying again...", retryInterval)
				<-time.After(retryInterval)
				continue
			}
			logrus.Infof("approval request has updated status[%v] (by %v)", approvalRequest.Approval.Status, approvalRequest.Approval.ApproverId)
			break
		}

		return nil
	},
}
