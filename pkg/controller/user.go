package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/common"
)

type ListUsersV1Output []ListUsersV1OutputUser

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

func (c Client) ListUsersV1() (*ListUsersV1Output, *http.Response, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/api/v1/users"
	httpRequest, err := http.NewRequest(
		http.MethodGet,
		controllerUrl.String(),
		nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create http request to create a session: %s", err)
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
		return nil, nil, fmt.Errorf("failed to execute http request to create session: %s", err)
	}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, httpResponse, fmt.Errorf("failed to read response body: %s", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return nil, httpResponse, fmt.Errorf("failed to receive a successful response (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, httpResponse, fmt.Errorf("failed to parse response from controller service: %s", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return nil, httpResponse, fmt.Errorf("failed to parse response data from controller service: %s", err)
	}
	var output ListUsersV1Output
	if err := json.Unmarshal(responseData, &output); err != nil {
		return nil, httpResponse, fmt.Errorf("failed to unmarshal response data into output: %s", err)
	}
	return &output, httpResponse, nil
}
