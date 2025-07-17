package automationtemplate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"opsicle/internal/approvals"
	"opsicle/internal/approver"
	"opsicle/internal/automations"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/worker"
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
		resourceIsSpecified, resourcePath, err := cli.GetFilePathFromArgs(args)
		if err != nil {
			return fmt.Errorf("failed to get file path from args['%s']: %s", strings.Join(args, "', '"), err)
		} else if !resourceIsSpecified {
			return fmt.Errorf("failed to receive required <path-to-automation>")
		}
		automationTemplate, err := automations.LoadAutomationTemplateFromFile(resourcePath)
		if err != nil {
			return fmt.Errorf("failed to load automation from path[%s]: %s", resourcePath, err)
		}
		o, _ := json.MarshalIndent(automationTemplate, "", "  ")
		logrus.Tracef("received automation template:\n%s", string(o))

		requesterId := viper.GetString("requester-id")
		requesterName := viper.GetString("requester-name")

		// resoulve approval policy

		var approvalPolicy *approvals.PolicySpec
		externalPoliciesPath := viper.GetString("approval-policies-path")
		if automationTemplate.Spec.ApprovalPolicy != nil {
			logrus.Infof("automation template has an approval policy, loading it now")
			o, _ := json.MarshalIndent(automationTemplate.Spec.ApprovalPolicy, "", "  ")
			logrus.Tracef("following is the loaded approval policy:\n%s", string(o))
			if automationTemplate.Spec.ApprovalPolicy.PolicyRef != nil {
				logrus.Infof("approval policy is referring to an existing policy")
				approvalPolicyName := *automationTemplate.Spec.ApprovalPolicy.PolicyRef
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
				automationTemplate.Spec.ApprovalPolicy.Spec = &approvalPolicyResource.Spec
			}
			approvalPolicy = automationTemplate.Spec.ApprovalPolicy.Spec

			owners := []string{}
			for _, owner := range automationTemplate.Spec.Metadata.Owners {
				owners = append(owners, fmt.Sprintf("%s (%s)", owner.Name, owner.Email))
			}
			approvalRequest := approvals.RequestSpec{
				Id: automationTemplate.Metadata.Name,
				Message: fmt.Sprintf(
					"Requesting to execute '%s' written by: %s\n\n%s",
					automationTemplate.Spec.Metadata.DisplayName,
					strings.Join(owners, ", "),
					automationTemplate.Spec.Metadata.Description,
				),
				RequesterId:   requesterId,
				RequesterName: requesterName,
			}
			if approvalPolicy.Slack != nil {
				approvalRequest.Slack = []approvals.SlackRequestSpec{*approvalPolicy.Slack}
			}
			if approvalPolicy.Telegram != nil {
				approvalRequest.Telegram = []approvals.TelegramRequestSpec{*approvalPolicy.Telegram}
			}

			approverUrlData := viper.GetString("approver-url")
			approverUrl, err := url.Parse(approverUrlData)
			if err != nil {
				return fmt.Errorf("failed to parse approverUrl[%s] as a url: %s", approverUrlData, err)
			}
			logrus.Infof("using approver service at url[%s]", approverUrl)

			serviceLogs := make(chan common.ServiceLog, 64)
			common.StartServiceLogLoop(serviceLogs)
			approvalRequestData, err := json.Marshal(approvalRequest)
			if err != nil {
				return fmt.Errorf("failed to marshal approval request: %s", err)
			}

			approverUrl.Path = "/approval-request"
			req, err := http.NewRequest(
				http.MethodPost,
				approverUrl.String(),
				bytes.NewBuffer(approvalRequestData),
			)
			if err != nil {
				return fmt.Errorf("failed to create request for approver service: %s", err)
			}
			common.AddHttpHeaders(req)
			client := common.NewHttpClient()
			res, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("failed to execute request to approver service: %s", err)
			}
			responseBody, err := io.ReadAll(res.Body)
			if err != nil {
				return fmt.Errorf("failed to read response from approver service: %s", err)
			}
			logrus.Debugf("received response: %s", string(responseBody))
			var response common.HttpResponse
			if err := json.Unmarshal(responseBody, &response); err != nil {
				return fmt.Errorf("failed to parse response from approver service: %s", err)
			}
			responseData, err := json.Marshal(response.Data)
			if err != nil {
				return fmt.Errorf("failed to reconcile response into data from approver service: %s", err)
			}
			var requestSpec approvals.RequestSpec
			if err := json.Unmarshal(responseData, &requestSpec); err != nil {
				return fmt.Errorf("failed to parse data from approver service: %s", err)
			}
			requestId := requestSpec.Id
			requestUuid := requestSpec.GetUuid()
			logrus.Infof("submitted request[%s:%s]", requestId, requestUuid)

			var approval approvals.ApprovalSpec
			isDone := false
			for !isDone {
				logrus.Infof("getting status from url[%s]...", approverUrl.String())
				approverUrl.Path = fmt.Sprintf("/approval-request/%v", requestUuid)
				req, err = http.NewRequest(
					http.MethodGet,
					approverUrl.String(),
					bytes.NewBuffer(approvalRequestData),
				)
				if err != nil {
					return fmt.Errorf("failed to create request for approver service: %s", err)
				}
				common.AddHttpHeaders(req)
				res, err = client.Do(req)
				if err != nil {
					return fmt.Errorf("failed to execute request to approver service: %s", err)
				}
				responseBody, err := io.ReadAll(res.Body)
				if err != nil {
					return fmt.Errorf("failed to read response from approver service: %s", err)
				}
				logrus.Debugf("received response from url[%s]: %s", approverUrl.String(), string(responseBody))
				var response common.HttpResponse
				if err := json.Unmarshal(responseBody, &response); err != nil {
					return fmt.Errorf("failed to parse response from approver service: %s", err)
				}
				responseData, err := json.Marshal(response.Data)
				if err != nil {
					return fmt.Errorf("failed to parse response from approver service: %s", err)
				}
				var approvalRequest approver.ApprovalRequest
				if err := json.Unmarshal(responseData, &approvalRequest); err != nil {
					return fmt.Errorf("failed to parse response from approver service: %s", err)
				}
				if approvalRequest.Spec.Approval == nil {
					logrus.Infof("request is still not yet approved, waiting another 5 seconds...")
					<-time.After(5 * time.Second)
					continue
				}
				logrus.Infof("approval request has updated status[%v] (by %v)", approvalRequest.Spec.Approval.Status, approvalRequest.Spec.Approval.ApproverId)
				approval = *approvalRequest.Spec.Approval
				isDone = true
			}

			if approval.Status != approvals.StatusApproved {
				logrus.Errorf("approval request was not approved")
				return fmt.Errorf("not proceeding, request was rejected")
			}
		}

		automationInstance := &automations.Automation{
			Resource: common.Resource{
				Metadata: common.Metadata{
					Name: automationTemplate.Metadata.Name,
				},
			},
			Spec: automationTemplate.Spec.Template,
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
			fmt.Println("----------------------------------------")
			fmt.Println("----------------------------------------")
			fmt.Println("----------------------------------------")
			fmt.Println("----------------------------------------")
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
