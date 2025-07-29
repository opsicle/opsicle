package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/controller/session"
)

type CreateSessionV1Opts struct {
	OrgCode  string `json:"orgCode"`
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

func (c Client) DeleteSessionV1() (string, *http.Response, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/api/v1/session"
	httpRequest, err := http.NewRequest(
		http.MethodDelete,
		controllerUrl.String(),
		nil,
	)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create http request to create a session: %s", err)
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
		return "", nil, fmt.Errorf("failed to execute http request to create session: %s", err)
	}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return "", httpResponse, fmt.Errorf("failed to read response body: %s", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return "", httpResponse, fmt.Errorf("failed to receive a successful response (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", httpResponse, fmt.Errorf("failed to parse response from controller service: %s", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return "", httpResponse, fmt.Errorf("failed to parse response data from controller service: %s", err)
	}
	return string(responseData), httpResponse, nil
}
