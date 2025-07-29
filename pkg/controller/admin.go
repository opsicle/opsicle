package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/common"
)

type InitV1Opts struct {
	AdminApiToken string `json:"-"`

	Email string `json:"email"`

	Password string `json:"password"`
}

type initV1EndpointResponse struct {
	UserId string `json:"userId"`
	OrgId  string `json:"orgId"`
}

func (c Client) InitV1(opts InitV1Opts) (userId, orgId string, err error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/admin/v1/init"
	requestBodyData, err := json.Marshal(opts)
	if err != nil {
		return "", "", fmt.Errorf("admin.InitV1: failed to marshal data: %s", err)
	}
	requestBody := bytes.NewBuffer(requestBodyData)
	httpRequest, err := http.NewRequest(
		http.MethodPost,
		controllerUrl.String(),
		requestBody,
	)
	if err != nil {
		return "", "", fmt.Errorf("admin.InitV1: failed to create http request: %s", err)
	}
	httpRequest.Header.Add("Content-Type", "application/json")
	httpRequest.Header.Add("User-Agent", fmt.Sprintf("opsicle/controller-sdk/client-%s", c.Id))
	httpRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", opts.AdminApiToken))
	httpResponse, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		return "", "", fmt.Errorf("admin.InitV1: failed to execute http request: %s", err)
	}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return "", "", fmt.Errorf("admin.InitV1: failed to read response body: %s", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("admin.InitV1: failed to receive a successful response (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", "", fmt.Errorf("admin.InitV1: failed to parse response from controller service: %s", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return "", "", fmt.Errorf("admin.InitV1: failed to parse response data from controller service: %s", err)
	}
	var endpointResponse initV1EndpointResponse
	if err := json.Unmarshal(responseData, &endpointResponse); err != nil {
		return "", "", fmt.Errorf("admin.InitV1: failed to derive business response data from controller: %s", err)
	}
	return endpointResponse.UserId, endpointResponse.OrgId, nil
}
