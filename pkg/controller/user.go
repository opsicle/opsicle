package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/common"
)

type CreateUserV1Input struct {
	// OrgInviteCode if present, allows the automatic registering of the user
	// with the organisation. Each invite code is only valid once
	OrgInviteCode *string `json:"orgInviteCode"`

	// Email is the user's email address
	Email string `json:"email"`

	// Password is the user's password
	Password string `json:"password"`
}

type CreateUserV1Output struct {
	Data CreateUserV1OutputData
	http.Response
}

type CreateUserV1OutputData struct {
	Id      string  `json:"id"`
	Email   string  `json:"email"`
	OrgCode *string `json:"orgCode"`
	OrgId   *string `json:"orgId"`
}

func (c Client) CreateUserV1(input CreateUserV1Input) (*CreateUserV1Output, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/api/v1/users"
	requestData, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input into json: %s", err)
	}
	requestBody := bytes.NewBuffer(requestData)
	httpRequest, err := http.NewRequest(
		http.MethodPost,
		controllerUrl.String(),
		requestBody,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request to create a session: %s", err)
	}
	httpRequest.Header.Add("Content-Type", "application/json")
	httpRequest.Header.Add("User-Agent", fmt.Sprintf("opsicle/controller-sdk/client-%s", c.Id))
	httpResponse, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request to create user: %s", err)
	}
	output := CreateUserV1Output{Response: *httpResponse}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return &output, fmt.Errorf("failed to read response body: %s", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return &output, fmt.Errorf("failed to receive a successful response (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return &output, fmt.Errorf("failed to parse response from controller service: %s", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return &output, fmt.Errorf("failed to parse response data from controller service: %s", err)
	}
	var data CreateUserV1OutputData
	if err := json.Unmarshal(responseData, &data); err != nil {
		return &output, fmt.Errorf("failed to unmarshal response data into output: %s", err)
	}
	output.Data = data
	return &output, nil
}

type ListUsersV1Output struct {
	Users []ListUsersV1OutputUser

	http.Response
}

type ListUsersV1OutputUser struct {
	Id    string                    `json:"id"`
	Email string                    `json:"email"`
	Org   *ListUsersV1OutputUserOrg `json:"org"`
	Type  string                    `json:"type"`
}

type ListUsersV1OutputUserOrg struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

func (c Client) ListUsersV1() (*ListUsersV1Output, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/api/v1/users"
	httpRequest, err := http.NewRequest(
		http.MethodGet,
		controllerUrl.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request to create a session: %s", err)
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
		return nil, fmt.Errorf("failed to execute http request to create session: %s", err)
	}
	output := ListUsersV1Output{Response: *httpResponse}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return &output, fmt.Errorf("failed to read response body: %s", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return &output, fmt.Errorf("failed to receive a successful response (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return &output, fmt.Errorf("failed to parse response from controller service: %s", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return &output, fmt.Errorf("failed to parse response data from controller service: %s", err)
	}
	if err := json.Unmarshal(responseData, &output.Users); err != nil {
		return &output, fmt.Errorf("failed to unmarshal response data into output: %s", err)
	}
	return &output, nil
}

type VerifyUserV1Input struct {
	Code string
}

type VerifyUserV1Output struct {
	Data VerifyUserV1OutputData

	http.Response
}

type VerifyUserV1OutputData struct {
	Email string `json:"email"`
}

func (c Client) VerifyUserV1(opts VerifyUserV1Input) (*VerifyUserV1Output, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = fmt.Sprintf("/api/v1/verification/%s", opts.Code)
	httpRequest, err := http.NewRequest(
		http.MethodGet,
		controllerUrl.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request to create a session: %s", err)
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
		return nil, fmt.Errorf("failed to execute http request to create session: %s", err)
	}
	output := VerifyUserV1Output{Response: *httpResponse}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return &output, fmt.Errorf("failed to read response body: %s", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return &output, fmt.Errorf("failed to receive a successful response (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return &output, fmt.Errorf("failed to parse response from controller service: %s", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return &output, fmt.Errorf("failed to parse response data from controller service: %s", err)
	}
	var data VerifyUserV1OutputData
	if err := json.Unmarshal(responseData, &data); err != nil {
		return &output, fmt.Errorf("failed to unmarshal response data into output: %s", err)
	}
	output.Data = data
	return &output, nil
}
