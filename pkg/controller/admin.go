package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/common"
)

type InitV1Input struct {
	AdminApiToken string `json:"-"`

	Email string `json:"email"`

	Password string `json:"password"`
}

type InitV1Output struct {
	Data InitV1OutputData

	http.Response
}

type InitV1OutputData struct {
	UserId string `json:"userId"`
	OrgId  string `json:"orgId"`
}

func (c Client) InitV1(opts InitV1Input) (*InitV1Output, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/admin/v1/init"
	requestBodyData, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("admin.InitV1: failed to marshal data: %w", err)
	}
	requestBody := bytes.NewBuffer(requestBodyData)
	httpRequest, err := http.NewRequest(
		http.MethodPost,
		controllerUrl.String(),
		requestBody,
	)
	if err != nil {
		return nil, fmt.Errorf("admin.InitV1: failed to create http request: %w", err)
	}
	httpRequest.Header.Add("Content-Type", "application/json")
	httpRequest.Header.Add("User-Agent", fmt.Sprintf("opsicle/controller-sdk/client-%s", c.Id))
	httpRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", opts.AdminApiToken))
	httpResponse, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("admin.InitV1: failed to execute http request: %w", err)
	}
	output := InitV1Output{Response: *httpResponse}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return &output, fmt.Errorf("admin.InitV1: failed to read response body: %w", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return &output, fmt.Errorf("admin.InitV1: failed to receive a successful response (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return &output, fmt.Errorf("admin.InitV1: failed to parse response from controller service: %w", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return &output, fmt.Errorf("admin.InitV1: failed to parse response data from controller service: %w", err)
	}
	var data InitV1OutputData
	if err := json.Unmarshal(responseData, &data); err != nil {
		return &output, fmt.Errorf("admin.InitV1: failed to derive business response data from controller: %w", err)
	}
	output.Data = data
	return &output, nil
}
