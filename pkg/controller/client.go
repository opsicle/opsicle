package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"opsicle/internal/common"
	"opsicle/internal/controller/session"
)

type NewClientOpts struct {
	ControllerUrl string
	BasicAuth     *NewClientBasicAuthOpts
	BearerAuth    *NewClientBearerAuthOpts
	Id            string
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

	controllerUrl, err := url.Parse(opts.ControllerUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse provided controllerUrl[%s]: %s", opts.ControllerUrl, err)
	}

	if controllerUrl.Scheme == "" {
		return nil, fmt.Errorf("failed to determine url scheme of controllerUrl[%s]", opts.ControllerUrl)
	}
	client.ControllerUrl = controllerUrl

	return client, nil
}

type Client struct {
	// ControllerUrl is the URL where the approver service is accessible
	// at
	ControllerUrl *url.URL
	BasicAuth     *NewClientBasicAuthOpts
	BearerAuth    *NewClientBearerAuthOpts

	// HttpClient is the HTTP client
	HttpClient *http.Client

	// Id will be included in the user-agent for identification
	Id string
}

type CreateSessionV1Opts struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (c Client) CreateSessionV1(opts CreateSessionV1Opts) (string, string, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/api/v1/session"
	requestBodyData, err := json.Marshal(opts)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal data: %s", err)
	}
	requestBody := bytes.NewBuffer(requestBodyData)
	httpRequest, err := http.NewRequest(
		http.MethodPost,
		controllerUrl.String(),
		requestBody,
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create http request to create a session: %s", err)
	}
	httpRequest.Header.Add("Content-Type", "application/json")
	httpRequest.Header.Add("User-Agent", fmt.Sprintf("opsicle/controller-sdk/client-%s", c.Id))
	if c.BasicAuth != nil {
		httpRequest.SetBasicAuth(c.BasicAuth.Username, c.BasicAuth.Password)
	}
	if c.BearerAuth != nil {
		httpRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.BearerAuth.Token))
	}
	httpResponse, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		return "", "", fmt.Errorf("failed to execute http request to create session: %s", err)
	}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %s", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to receive a successful response (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", "", fmt.Errorf("failed to parse response from controller service: %s", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse response data from controller service: %s", err)
	}
	var token session.Token
	if err := json.Unmarshal(responseData, &token); err != nil {
		return "", "", fmt.Errorf("failed to parse response from controller service into a session token: %s", err)
	}
	return token.Id, token.Value, nil
}

func (c Client) DeleteSessionV1() (string, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/api/v1/session"
	httpRequest, err := http.NewRequest(
		http.MethodDelete,
		controllerUrl.String(),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create http request to create a session: %s", err)
	}
	httpRequest.Header.Add("Content-Type", "application/json")
	httpRequest.Header.Add("User-Agent", fmt.Sprintf("opsicle/controller-sdk/client-%s", c.Id))
	if c.BasicAuth != nil {
		httpRequest.SetBasicAuth(c.BasicAuth.Username, c.BasicAuth.Password)
	}
	if c.BearerAuth != nil {
		httpRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.BearerAuth.Token))
	}
	httpResponse, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		return "", fmt.Errorf("failed to execute http request to create session: %s", err)
	}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %s", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to receive a successful response (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse response from controller service: %s", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return "", fmt.Errorf("failed to parse response data from controller service: %s", err)
	}
	return string(responseData), nil
}
