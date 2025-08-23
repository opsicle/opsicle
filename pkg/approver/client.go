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
)

type NewClientOpts struct {
	ApproverUrl string
	BasicAuth   *NewClientBasicAuthOpts
	BearerAuth  *NewClientBearerAuthOpts
	Id          string
}

type NewClientBasicAuthOpts struct {
	Username string
	Password string
}

type NewClientBearerAuthOpts struct {
	Token string
}

func NewClient(opts NewClientOpts) (*Client, error) {
	client := &Client{
		BasicAuth:  opts.BasicAuth,
		BearerAuth: opts.BearerAuth,
		HttpClient: &http.Client{},
		Id:         opts.Id,
	}

	approverUrl, err := url.Parse(opts.ApproverUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse provided ApproverUrl[%s]: %s", opts.ApproverUrl, err)
	}

	if approverUrl.Scheme == "" {
		return nil, fmt.Errorf("failed to determine url scheme of ApprovalUrl[%s]", opts.ApproverUrl)
	}
	client.ApproverUrl = approverUrl

	return client, nil
}

type Client struct {
	// ApproverUrl is the URL where the approver service is accessible
	// at
	ApproverUrl *url.URL
	BasicAuth   *NewClientBasicAuthOpts
	BearerAuth  *NewClientBearerAuthOpts

	// HttpClient is the HTTP client
	HttpClient *http.Client

	// Id will be included in the user-agent for identification
	Id string
}

// CreateApprovalRequest sends a message to the approver service and
// returns the UUID of the request issued by the approver service
func (c *Client) CreateApprovalRequest(input CreateApprovalRequestInput) (requestUuid string, err error) {
	approvalRequest := approvals.RequestSpec{
		Id:            input.Id,
		Links:         input.Links,
		Message:       input.Message,
		RequesterId:   input.RequesterId,
		RequesterName: input.RequesterName,
		Slack:         input.Slack,
		Telegram:      input.Telegram,
	}
	approvalRequestData, err := json.Marshal(approvalRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal approval request: %w", err)
	}
	approverUrl := *c.ApproverUrl
	approverUrl.Path = "/api/v1/approval-request"
	httpRequest, err := http.NewRequest(
		http.MethodPost,
		approverUrl.String(),
		bytes.NewBuffer(approvalRequestData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create http request for request[%s]: %s", approvalRequest.Id, err)
	}
	httpRequest.Header.Add("Content-Type", "application/json")
	httpRequest.Header.Add("User-Agent", fmt.Sprintf("opsicle-sdk/client-%s", c.Id))
	if c.BasicAuth != nil {
		httpRequest.SetBasicAuth(c.BasicAuth.Username, c.BasicAuth.Password)
	}
	if c.BearerAuth != nil {
		httpRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.BearerAuth.Token))
	}
	httpResponse, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		return "", fmt.Errorf("failed to execute http request for request[%s]: %s", approvalRequest.Id, err)
	}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to receive a successful response (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse response from approver service: %w", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return "", fmt.Errorf("failed to parse response data from approver service: %w", err)
	}
	if err := json.Unmarshal(responseData, &approvalRequest); err != nil {
		return "", fmt.Errorf("failed to parse response from approver service: %w", err)
	}
	return approvalRequest.GetUuid(), nil
}

func (c *Client) GetApproval(approvalUuid string) (*approvals.ApprovalSpec, error) {
	approverUrl := *c.ApproverUrl
	approverUrl.Path = fmt.Sprintf("/api/v1/approval/%s", approvalUuid)
	httpRequest, err := http.NewRequest(
		http.MethodGet,
		approverUrl.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request to get approval[%s]: %s", approvalUuid, err)
	}
	httpRequest.Header.Add("Content-Type", "application/json")
	httpRequest.Header.Add("User-Agent", fmt.Sprintf("opsicle-sdk/client-%s", c.Id))
	if c.BasicAuth != nil {
		httpRequest.SetBasicAuth(c.BasicAuth.Username, c.BasicAuth.Password)
	}
	if c.BearerAuth != nil {
		httpRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.BearerAuth.Token))
	}
	httpResponse, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request to get approval[%s]: %s", approvalUuid, err)
	}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to receive a successful response (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response from approver service: %w", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response data from approver service: %w", err)
	}
	var approval approvals.ApprovalSpec
	if err := json.Unmarshal(responseData, &approval); err != nil {
		return nil, fmt.Errorf("failed to parse response from approver service into an approval: %w", err)
	}
	return &approval, nil
}

func (c *Client) GetApprovalRequest(requestUuid string) (*approvals.RequestSpec, error) {
	approverUrl := *c.ApproverUrl
	approverUrl.Path = fmt.Sprintf("/api/v1/approval-request/%s", requestUuid)
	httpRequest, err := http.NewRequest(
		http.MethodGet,
		approverUrl.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request to get request[%s]: %s", requestUuid, err)
	}
	httpRequest.Header.Add("Content-Type", "application/json")
	httpRequest.Header.Add("User-Agent", fmt.Sprintf("opsicle-sdk/client-%s", c.Id))
	if c.BasicAuth != nil {
		httpRequest.SetBasicAuth(c.BasicAuth.Username, c.BasicAuth.Password)
	}
	if c.BearerAuth != nil {
		httpRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.BearerAuth.Token))
	}
	httpResponse, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request to get request[%s]: %s", requestUuid, err)
	}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to receive a successful response (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response from approver service: %w", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response data from approver service: %w", err)
	}
	var approvalRequest approvals.Request
	if err := json.Unmarshal(responseData, &approvalRequest); err != nil {
		return nil, fmt.Errorf("failed to parse response from approver service into an approval: %w", err)
	}
	return &approvalRequest.Spec, nil

}

func (c *Client) ListApprovalRequests() ([]string, error) {
	approverUrl := *c.ApproverUrl
	approverUrl.Path = "/api/v1/approval-request"
	httpRequest, err := http.NewRequest(
		http.MethodGet,
		approverUrl.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request to get approval requests: %w", err)
	}
	httpRequest.Header.Add("Content-Type", "application/json")
	httpRequest.Header.Add("User-Agent", fmt.Sprintf("opsicle-sdk/client-%s", c.Id))
	if c.BasicAuth != nil {
		httpRequest.SetBasicAuth(c.BasicAuth.Username, c.BasicAuth.Password)
	}
	if c.BearerAuth != nil {
		httpRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.BearerAuth.Token))
	}
	httpResponse, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request to get approval requests: %w", err)
	}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to receive a successful response (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response from approver service: %w", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response data from approver service: %w", err)
	}
	var approvalRequestKeys []string
	if err := json.Unmarshal(responseData, &approvalRequestKeys); err != nil {
		return nil, fmt.Errorf("failed to parse response from approver service into an approval: %w", err)
	}
	return approvalRequestKeys, nil
}
