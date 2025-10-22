package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/audit"
	"opsicle/internal/common"
	"opsicle/internal/types"
	"time"
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
	var outputData CreateUserV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPost,
		Path:   "/api/v1/users",
		Data:   input,
		Output: &outputData,
	})
	var output *CreateUserV1Output = nil
	if !errors.Is(err, types.ErrorOutputNil) {
		output = &CreateUserV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case types.ErrorEmailExists.Error():
			err = types.ErrorEmailExists
		}
	}
	return output, err
}

type ListUserAuditLogsV1Output struct {
	Data ListUserAuditLogsV1OutputData

	http.Response
}

type ListUserAuditLogsV1OutputData audit.LogEntries

type ListUserAuditLogsV1Input struct {
	Cursor  time.Time `json:"cursor"`
	Limit   int64     `json:"limit"`
	Reverse bool      `json:"reverse"`
}

func (c Client) ListUserAuditLogsV1(input ListUserAuditLogsV1Input) (*ListUserAuditLogsV1Output, error) {
	var outputData ListUserAuditLogsV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   "/api/v1/user/logs",
		Data:   input,
		Output: &outputData,
	})
	var output *ListUserAuditLogsV1Output = nil
	if !errors.Is(err, types.ErrorOutputNil) {
		output = &ListUserAuditLogsV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	return output, err
}

type ListUsersV1Output struct {
	Data ListUsersV1OutputData

	http.Response
}

type ListUsersV1OutputData struct {
	Users []ListUsersV1OutputUser
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
		return nil, fmt.Errorf("failed to create http request to create a session: %w", err)
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
		return nil, fmt.Errorf("failed to execute http request to create session: %w", err)
	}
	output := ListUsersV1Output{Response: *httpResponse}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return &output, fmt.Errorf("failed to read response body: %w", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return &output, fmt.Errorf("failed to receive a successful response (status code: %v): %s", httpResponse.StatusCode, string(responseBody))
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return &output, fmt.Errorf("failed to parse response from controller service: %w", err)
	}
	responseData, err := json.Marshal(response.Data)
	if err != nil {
		return &output, fmt.Errorf("failed to parse response data from controller service: %w", err)
	}
	var data ListUsersV1OutputData
	if err := json.Unmarshal(responseData, &data); err != nil {
		return &output, fmt.Errorf("failed to unmarshal response data into output: %w", err)
	}
	output.Data = data
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
	var outputData VerifyUserV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/api/v1/verification/%s", opts.Code),
		Output: &outputData,
	})
	var output *VerifyUserV1Output = nil
	if !errors.Is(err, types.ErrorOutputNil) {
		output = &VerifyUserV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	return output, err
}
