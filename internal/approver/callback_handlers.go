package approver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"opsicle/internal/approvals"
	"opsicle/internal/common"
	"time"
)

func getWebhookCallbackClient() *http.Client {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	return &client
}

func getWebhookCallbackRequest(opts approvals.WebhookCallbackSpec) (*http.Request, error) {
	targetUrl, err := url.Parse(opts.Url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse targetUrl[%s]: %s", targetUrl, err)
	}
	method := opts.Method
	if method == "" {
		method = http.MethodPost
	}
	req := http.Request{
		Method: method,
		URL:    targetUrl,
	}
	if opts.Auth != nil {
		if opts.Auth.Basic != nil {
			username := opts.Auth.Basic.Username
			password := opts.Auth.Basic.Password
			req.SetBasicAuth(username, password)
		}
		if opts.Auth.Bearer != nil {
			bearerToken := opts.Auth.Bearer.Value
			req.Header.Add("Authorization", "Bearer "+bearerToken)
		}
		if opts.Auth.Header != nil {
			authHeader := opts.Auth.Header
			req.Header.Add(authHeader.Key, authHeader.Value)
		}
	}
	return &req, nil
}

type handleCallbackOpts struct {
	Req         *ApprovalRequest
	ServiceLogs chan<- common.ServiceLog
}

// handleCallback is the entrypoint for handling the `approvals.CallbackSpec`
// specification
//
// doctags: #callback-type-priority
func handleCallback(opts handleCallbackOpts) error {
	requestId := opts.Req.Spec.Id
	requestUuid := opts.Req.Spec.GetUuid()
	if opts.Req.Spec.Callback == nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "no callback action was found for request[%s:%s]", requestId, requestUuid)
		return nil
	}
	callback := opts.Req.Spec.Callback
	callbackType := callback.Type
	switch callbackType {
	case approvals.CallbackWebhook:
		break
	default:
		if callback.Webhook != nil {
			callbackType = approvals.CallbackWebhook
		}
	}
	switch callbackType {
	case approvals.CallbackWebhook:
		if err := handleWebhookCallback(opts); err != nil {
			return fmt.Errorf("failed to process webhook callback: %w", err)
		}
	default:
		return fmt.Errorf("failed to handle callback of type[%s]", callback.Type)
	}
	return nil
}

func handleWebhookCallback(opts handleCallbackOpts) error {
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "processing webhook callback for request[%s:%s]", opts.Req.Spec.Id, opts.Req.Spec.GetUuid())

	// sanity checks

	if opts.Req.Spec.Callback.Webhook == nil {
		return fmt.Errorf("failed to process webhook callback: no webhook specification found")
	}
	if opts.Req.Spec.Approval == nil {
		return fmt.Errorf("failed to receive a valid approval specification")
	}
	webhook := opts.Req.Spec.Callback.Webhook
	webhookClient := getWebhookCallbackClient()
	webhookRequest, err := getWebhookCallbackRequest(*webhook)
	if err != nil {
		return fmt.Errorf("failed to create webhook callback request: %w", err)
	}
	approvalData, _ := json.Marshal(opts.Req.Spec.Approval)
	webhookRequest.Body = io.NopCloser(bytes.NewBuffer(approvalData))

	// setup for the retry loop

	retryCount := 5
	if webhook.RetryCount != nil && *webhook.RetryCount > 0 {
		retryCount = *webhook.RetryCount
	}
	isExponentialBackoffEnabled := false
	retryIntervalSeconds := -1
	if webhook.RetryIntervalSeconds != nil && *webhook.RetryIntervalSeconds > 0 {
		retryIntervalSeconds = *webhook.RetryIntervalSeconds
	}
	if retryIntervalSeconds == -1 {
		retryIntervalSeconds = 5
		isExponentialBackoffEnabled = true
	}
	isRequestSuccessful := false
	currentRetryAttempt := 1

	// retry loop begins

	startedAt := time.Now()
	for !isRequestSuccessful {
		webhookResponse, err := webhookClient.Do(webhookRequest)
		if err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelWarn, "attempt[%v/%v] failed to execute webhook callback request: %s", currentRetryAttempt, retryCount, err)
		} else if webhookResponse.StatusCode != http.StatusOK {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelWarn, "attempt[%v/%v] failed to receive status code %v", currentRetryAttempt, retryCount, http.StatusOK)
		} else {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "attempt[%v/%v] received status code %v from url[%s]", currentRetryAttempt, retryCount, http.StatusOK, webhook.Url)
			isRequestSuccessful = true
			break
		}
		if retryCount == currentRetryAttempt {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "attempt[%v/%v] retries exhausted, stopping...", currentRetryAttempt, retryCount)
			break
		}
		if isExponentialBackoffEnabled {
			retryIntervalSeconds += retryIntervalSeconds
		}
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "attempt[%v/%v] failed, next retry happening in %v seconds...", currentRetryAttempt, retryCount, retryIntervalSeconds)
		currentRetryAttempt++
		<-time.After(time.Duration(retryIntervalSeconds) * time.Second)
	}
	if !isRequestSuccessful {
		return fmt.Errorf("failed to execute webhook callback after %v attempts", retryCount)
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "successfully processed webhook callback for request[%s:%s] after %s", opts.Req.Spec.Id, opts.Req.Spec.GetUuid(), time.Since(startedAt))
	return nil
}
