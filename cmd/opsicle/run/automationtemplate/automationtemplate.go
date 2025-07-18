package automationtemplate

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/approvals"
	"opsicle/internal/automations"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/worker"
	approverApi "opsicle/pkg/approver"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "approval-policies-path",
		Short:        'p',
		DefaultValue: "",
		Usage:        "specifies the directory to where ApprovalPolicy resources can be found",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "approver-retry-interval",
		DefaultValue: 5 * time.Second,
		Usage:        "defines the retry interval for retrieving the status",
		Type:         cli.FlagTypeDuration,
	},
	{
		Name:         "approver-url",
		DefaultValue: "http://localhost:12345",
		Usage:        "defines the url where the approver service is accessible at if an ApprovalPolicy is defined",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "requester-id",
		DefaultValue: "opsicle-run-automation-template-user",
		Usage:        "defines the ID of the user who will be executing the AutomationTemplate reosurce",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "requester-name",
		DefaultValue: "Opsicle Dev User",
		Usage:        "defines the name of the user who will be executing the AutomationTemplate reosurce",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "automationtemplate",
	Aliases: []string{"at"},
	Short:   "Runs an AutomationTemplate resource independently",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		resourcePath, err := cli.GetFilePathFromArgs(args)
		if err != nil {
			return fmt.Errorf("failed to receive required <path-to-automation>: %s", err)
		}
		automationTemplateInstance, err := automations.LoadAutomationTemplateFromFile(resourcePath)
		if err != nil {
			return fmt.Errorf("failed to load automation template from path[%s]: %s", resourcePath, err)
		}
		o, _ := json.MarshalIndent(automationTemplateInstance, "", "  ")
		logrus.Debugf("loaded automation template as follows:\n%s", string(o))

		requesterId := viper.GetString("requester-id")
		requesterName := viper.GetString("requester-name")

		// resolve approval policy and get approval if needed

		var approvalPolicy *approvals.PolicySpec
		externalPoliciesPath := viper.GetString("approval-policies-path")
		if automationTemplateInstance.Spec.ApprovalPolicy != nil {
			logrus.Infof("automation template has an approval policy, loading it now")
			o, _ := json.MarshalIndent(automationTemplateInstance.Spec.ApprovalPolicy, "", "  ")
			logrus.Tracef("following is the loaded approval policy:\n%s", string(o))
			if automationTemplateInstance.Spec.ApprovalPolicy.PolicyRef != nil {
				logrus.Infof("approval policy is referring to an existing policy")
				approvalPolicyName := *automationTemplateInstance.Spec.ApprovalPolicy.PolicyRef
				if externalPoliciesPath == "" {
					return fmt.Errorf("failed to receive a path where approval policies can be found but policyRef was defined")
				}
				matchedResources, err := common.FindResourceInFilesystem(externalPoliciesPath, approvalPolicyName, "ApprovalPolicy")
				if err != nil {
					return fmt.Errorf("failed while finding resource of type[ApprovalPolicy] with name[%s]: %s", approvalPolicyName, err)
				}
				if len(matchedResources) <= 0 {
					return fmt.Errorf("failed to get any resources of type[ApprovalPolicy] with name[%s]: %s", approvalPolicyName, err)
				} else if len(matchedResources) > 1 {
					weirdFiles := []string{}
					for _, matchedResource := range matchedResources {
						weirdFiles = append(weirdFiles, matchedResource.Path)
					}
					return fmt.Errorf("more than one resource had type[ApprovalPolicy] with name[%s]: ['%s']", approvalPolicyName, strings.Join(weirdFiles, "', '"))
				}
				logrus.Tracef("following is the data from the matched resource:\n%s", string(matchedResources[0].Data))
				var approvalPolicyResource approvals.Policy
				if err := yaml.Unmarshal(matchedResources[0].Data, &approvalPolicyResource); err != nil {
					return fmt.Errorf("failed to parse file[%s]: %s", matchedResources[0].Path, err)
				}
				automationTemplateInstance.Spec.ApprovalPolicy.Spec = &approvalPolicyResource.Spec
			}
			approvalPolicy = automationTemplateInstance.Spec.ApprovalPolicy.Spec

			owners := []string{}
			for _, owner := range automationTemplateInstance.Spec.Metadata.Owners {
				owners = append(owners, fmt.Sprintf("%s (%s)", owner.Name, owner.Email))
			}
			approvalRequestInstance := approvals.RequestSpec{
				Id: automationTemplateInstance.Metadata.Name,
				Message: fmt.Sprintf(
					"Requesting to execute '%s' written by: %s\n\n%s",
					automationTemplateInstance.Spec.Metadata.DisplayName,
					strings.Join(owners, ", "),
					automationTemplateInstance.Spec.Metadata.Description,
				),
				RequesterId:   requesterId,
				RequesterName: requesterName,
			}
			if approvalPolicy.Slack != nil {
				approvalRequestInstance.Slack = []approvals.SlackRequestSpec{*approvalPolicy.Slack}
			}
			if approvalPolicy.Telegram != nil {
				approvalRequestInstance.Telegram = []approvals.TelegramRequestSpec{*approvalPolicy.Telegram}
			}

			approverUrl := viper.GetString("approver-url")
			logrus.Infof("using approver service at url[%s]", approverUrl)

			serviceLogs := make(chan common.ServiceLog, 64)
			common.StartServiceLogLoop(serviceLogs)

			client, err := approverApi.NewClient(approverApi.NewClientOpts{
				ApproverUrl: approverUrl,
				Id:          "opsicle-run-approval",
			})
			if err != nil {
				return fmt.Errorf("failed to create client for approver service: %s", err)
			}
			requestUuid, err := client.CreateApprovalRequest(approverApi.CreateApprovalRequestInput{
				Callback:      approvalRequestInstance.Callback,
				Id:            approvalRequestInstance.Id,
				Links:         approvalRequestInstance.Links,
				Message:       approvalRequestInstance.Message,
				RequesterId:   approvalRequestInstance.RequesterId,
				RequesterName: approvalRequestInstance.RequesterName,
				Slack:         approvalRequestInstance.Slack,
				Telegram:      approvalRequestInstance.Telegram,
			})
			if err != nil {
				return fmt.Errorf("failed to create approval request: %s", err)
			}
			logrus.Infof("submitted request[%s]", requestUuid)
			retryInterval := viper.GetDuration("approver-retry-interval")
			logrus.Infof("checks will be done at %v intervals, set log level to debug to see intervals if needed", retryInterval)

			var approval approvals.ApprovalSpec
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
				approval = *approvalRequest.Approval
				logrus.Infof("approval request has updated status[%v] (by %v)", approvalRequest.Approval.Status, approvalRequest.Approval.ApproverId)
				break
			}

			if approval.Status != approvals.StatusApproved {
				logrus.Errorf("approval request was not approved")
				return fmt.Errorf("not proceeding, request was rejected")
			}
		}

		// start the automation

		automationInstance := &automations.Automation{
			Resource: common.Resource{
				Metadata: common.Metadata{
					Name: automationTemplateInstance.Metadata.Name,
				},
			},
			Spec: automationTemplateInstance.Spec.Template,
		}

		var logsWaiter sync.WaitGroup
		serviceLogs := make(chan common.ServiceLog, 64)
		automationLogs := make(chan string, 64)
		doneEventChannel := make(chan common.Done)
		logsWaiter.Add(1)
		go func() {
			<-doneEventChannel
			close(serviceLogs)
		}()
		logsWaiter.Add(1)
		go func() {
			// wait for the logs to finish, otherwise some logs
			// might not be printed
			defer logsWaiter.Done()
			for {
				automationLog, ok := <-automationLogs
				if !ok {
					break
				}
				fmt.Print(automationLog)
			}
		}()
		go func() {
			defer close(automationLogs)
			logrus.Infof("started worker logging event loop for automation[%s]", automationInstance.Resource.Metadata.Name)
			for {
				workerLog, ok := <-serviceLogs
				if !ok {
					break
				}
				logger := logrus.Info
				switch workerLog.Level {
				case common.LogLevelTrace:
					logger = logrus.Trace
				case common.LogLevelDebug:
					logger = logrus.Debug
				case common.LogLevelInfo:
					logger = logrus.Info
				case common.LogLevelWarn:
					logger = logrus.Warn
				case common.LogLevelError:
					logger = logrus.Error
				}
				logger(workerLog.Message)
			}
			logrus.Infof("worker logs have stopped streaming for automation[%s]", automationInstance.Resource.Metadata.Name)
			logsWaiter.Done()
		}()
		logsWaiter.Add(1)
		if err := worker.RunAutomation(worker.RunAutomationOpts{
			Done:           &doneEventChannel,
			Spec:           automationInstance,
			ServiceLogs:    serviceLogs,
			AutomationLogs: automationLogs,
		}); err != nil {
			return fmt.Errorf("automation execution failed with message: %s", err)
		}

		return nil
	},
}
