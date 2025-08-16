package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"
)

const (
	MfaTypeTotp = "totp"
)

type ListUserMfasV1Input struct{}

type ListUserMfasV1Output struct {
	Data ListUserMfasV1OutputData
	http.Response
}

type ListUserMfasV1OutputData []models.UserMfa

func (c Client) ListUserMfasV1(opts ListUserMfasV1Input) (*ListUserMfasV1Output, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/api/v1/user/mfas"
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
		return nil, fmt.Errorf("failed to execute http request to create user: %s", err)
	}
	output := ListUserMfasV1Output{Response: *httpResponse}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return &output, fmt.Errorf("failed to read response body: %s", err)
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return &output, fmt.Errorf("failed to parse response from controller service: %s", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return &output, fmt.Errorf("failed to parse response data from controller service: %s", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		err := fmt.Errorf("%v", response.Data)
		return &output, fmt.Errorf("failed to receive a successful response (status code: %v): %w", httpResponse.StatusCode, err)
	}
	var data ListUserMfasV1OutputData
	if err := json.Unmarshal(responseData, &data); err != nil {
		return &output, fmt.Errorf("failed to unmarshal response data into output: %s", err)
	}
	output.Data = data
	return &output, nil
}

type ListAvailableMfaTypesOutput struct {
	Data []ListAvailableMfaTypesOutputType `json:"data"`

	http.Response
}

type ListAvailableMfaTypesOutputType struct {
	Description string `json:"description"`
	Label       string `json:"label"`
	Value       string `json:"value"`
}

func (c Client) ListAvailableMfaTypes() (*ListAvailableMfaTypesOutput, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/api/v1/user/mfas"
	httpRequest, err := http.NewRequest(
		http.MethodOptions,
		controllerUrl.String(),
		nil,
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
	output := ListAvailableMfaTypesOutput{Response: *httpResponse}
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
	if err := json.Unmarshal(responseData, &output.Data); err != nil {
		return &output, fmt.Errorf("failed to unmarshal response data into output: %s", err)
	}
	return &output, nil
}

type CreateUserMfaV1Input struct {
	Password string `json:"password"`
	MfaType  string `json:"mfaType"`
}

type CreateUserMfaV1Output struct {
	Data CreateUserMfaV1OutputData

	http.Response
}

type CreateUserMfaV1OutputData struct {
	Id        string `json:"id"`
	Secret    string `json:"secret"`
	Type      string `json:"type"`
	UserEmail string `json:"userEmail"`
	UserId    string `json:"userId"`
}

func (c Client) CreateUserMfaV1(input CreateUserMfaV1Input) (*CreateUserMfaV1Output, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/api/v1/user/mfa"
	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("controller.CreateUserMfaV1: failed to marshal input into json: %w", err)
	}
	requestBody := bytes.NewBuffer(data)
	httpRequest, err := http.NewRequest(
		http.MethodPost,
		controllerUrl.String(),
		requestBody,
	)
	if err != nil {
		return nil, fmt.Errorf("controller.CreateUserMfaV1: failed to create http request: %w", err)
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
		return nil, fmt.Errorf("controller.CreateUserMfaV1: failed to execute http request: %w", err)
	}
	output := CreateUserMfaV1Output{Response: *httpResponse}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return &output, fmt.Errorf("controller.CreateUserMfaV1: failed to read response body: %w", err)
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return &output, fmt.Errorf("controller.CreateUserMfaV1: failed to parse response body: %w", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return &output, fmt.Errorf("controller.CreateUserMfaV1: failed to parse response body data: %w", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return &output, fmt.Errorf("%s", string(responseData))
	}
	if err := json.Unmarshal(responseData, &output.Data); err != nil {
		return &output, fmt.Errorf("controller.CreateUserMfaV1: failed to unmarshal response data into output: %w", err)
	}
	return &output, nil
}

type VerifyUserMfaV1Input struct {
	Id    string `json:"-"`
	Value string `json:"value"`
}

type VerifyUserMfaV1Output struct {
	http.Response
}

func (c Client) VerifyUserMfaV1(input VerifyUserMfaV1Input) (*VerifyUserMfaV1Output, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = fmt.Sprintf("/api/v1/user/mfa/%v", input.Id)
	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("controller.VerifyUserMfaV1: failed to marshal input into json: %w", err)
	}
	requestBody := bytes.NewBuffer(data)
	httpRequest, err := http.NewRequest(
		http.MethodPost,
		controllerUrl.String(),
		requestBody,
	)
	if err != nil {
		return nil, fmt.Errorf("controller.VerifyUserMfaV1: failed to create http request: %w", err)
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
		return nil, fmt.Errorf("controller.VerifyUserMfaV1: failed to execute http request: %w", err)
	}
	output := VerifyUserMfaV1Output{Response: *httpResponse}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return &output, fmt.Errorf("controller.VerifyUserMfaV1: failed to read response body: %w", err)
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return &output, fmt.Errorf("controller.VerifyUserMfaV1: failed to parse response body: %w", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return &output, fmt.Errorf("controller.VerifyUserMfaV1: failed to received a success status code (status code: %v): %w", httpResponse.StatusCode, err)
	}
	return &output, nil
}
