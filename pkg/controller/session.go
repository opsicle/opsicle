package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"
	"time"
)

type CreateSessionV1Input struct {
	OrgCode  *string `json:"orgCode"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Hostname string  `json:"hostname"`
}

type CreateSessionV1Output struct {
	SessionId    string
	SessionToken string

	http.Response
}

func (c Client) CreateSessionV1(opts CreateSessionV1Input) (*CreateSessionV1Output, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/api/v1/session"
	requestBodyData, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input data: %s", err)
	}
	requestBody := bytes.NewBuffer(requestBodyData)
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
	output := &CreateSessionV1Output{Response: *httpResponse}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return output, fmt.Errorf("failed to read response body: %s", err)
	}
	switch httpResponse.StatusCode {
	case http.StatusBadRequest:
		return output, fmt.Errorf("user credentials failed: %w", ErrorUserLoginFailed)
	case http.StatusLocked:
		return output, fmt.Errorf("user email is not verified: %w", ErrorUserEmailNotVerified)
	case http.StatusInternalServerError:
		return output, fmt.Errorf("received an unknown error (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return output, fmt.Errorf("failed to parse response from controller service: %s", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return output, fmt.Errorf("failed to parse response data from controller service: %s", err)
	}
	var token models.SessionToken
	if err := json.Unmarshal(responseData, &token); err != nil {
		return output, fmt.Errorf("failed to parse response from controller service into a session token: %s", err)
	}
	output.SessionId = token.SessionId
	output.SessionToken = token.Value
	return output, nil
}

type DeleteSessionV1Output struct {
	Data DeleteSessionV1OutputData

	http.Response
}

type DeleteSessionV1OutputData struct {
	// SessionId is only populated if the call to the controller was
	// successful as indicated by the `.IsSuccessful` property
	SessionId string `json:"sessionId"`

	// IsSuccessful indicates whether a session was deleted
	IsSuccessful bool `json:"isSuccessful"`
}

func (c Client) DeleteSessionV1() (*DeleteSessionV1Output, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/api/v1/session"
	httpRequest, err := http.NewRequest(
		http.MethodDelete,
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
	output := DeleteSessionV1Output{Response: *httpResponse}
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
	var data DeleteSessionV1OutputData
	if err := json.Unmarshal(responseData, &data); err != nil {
		return &output, fmt.Errorf("failed to unmarshal response data into expected output: %s", err)
	}
	output.Data = data
	return &output, nil
}

type ValidateSessionV1Output struct {
	Data ValidateSessionV1OutputData

	http.Response
}

type ValidateSessionV1OutputData struct {
	IsExpired bool      `json:"isExpired"`
	ExpiresAt time.Time `json:"expiresAt"`
	StartedAt time.Time `json:"startedAt"`
	UserId    string    `json:"userId"`
	Username  string    `json:"username"`
	UserType  string    `json:"userType"`
	OrgCode   *string   `json:"orgCode"`
	OrgId     *string   `json:"orgId"`
}

func (c Client) ValidateSessionV1() (*ValidateSessionV1Output, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/api/v1/session"
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
	httpRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.BearerAuth.Token))
	httpResponse, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request to create session: %s", err)
	}
	output := ValidateSessionV1Output{Response: *httpResponse}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return &output, fmt.Errorf("failed to read response body: %s", err)
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return &output, fmt.Errorf("failed to parse response from controller service: %s", err)
	}
	if output.Response.StatusCode != http.StatusOK {
		var err error
		switch response.Data.(string) {
		case ErrorAuthRequired.Error():
			err = ErrorAuthRequired
		default:
			err = ErrorGeneric
		}
		return &output, err
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return &output, fmt.Errorf("failed to parse response data from controller service: %s", err)
	}
	if err := json.Unmarshal(responseData, &output.Data); err != nil {
		return &output, fmt.Errorf("failed to parse final response data (%s) from controller service: %s", string(responseData), err)
	}
	output.Data.IsExpired = output.Data.ExpiresAt.Before(time.Now())
	return &output, nil
}
