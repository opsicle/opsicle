package approval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"opsicle/internal/approvals"
	"opsicle/internal/approver"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"os"
	"sync"
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
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "approval",
	Short: "Runs an approval manifest",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		resourceIsSpecified := false
		resourcePath := ""
		if len(args) > 0 {
			resourcePath = args[0]
			resourceIsSpecified = true
		}
		if !resourceIsSpecified {
			return fmt.Errorf("failed to receive a <path-to-template-file")
		}
		fi, err := os.Stat(resourcePath)
		if err != nil {
			return fmt.Errorf("failed to check for existence of file at path[%s]: %s", resourcePath, err)
		}
		if fi.IsDir() {
			return fmt.Errorf("failed to get a file at path[%s]: got a directory", resourcePath)
		}
		approvalRequestInstance, err := approvals.LoadRequestFromFile(resourcePath)
		if err != nil {
			return fmt.Errorf("failed to load approval request: %s", err)
		}
		o, _ := json.MarshalIndent(approvalRequestInstance, "", "  ")
		logrus.Infof("loaded approval request as follows:\n%s", string(o))

		approverUrlData := viper.GetString("approver-url")
		approverUrl, err := url.Parse(approverUrlData)
		if err != nil {
			return fmt.Errorf("failed to parse approverUrl[%s] as a url: %s", approverUrlData, err)
		}
		logrus.Infof("using approver service at url[%s]", approverUrl)

		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)
		approvalRequest := approver.ApprovalRequest{
			Spec: approvalRequestInstance.Spec,
		}
		approvalRequestData, err := json.Marshal(approvalRequest.Spec)
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
			isDone = true
		}

		var waiter sync.WaitGroup

		waiter.Wait()

		return nil
	},
}
