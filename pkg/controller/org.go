package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/common"
	"time"
)

type CreateOrgV1Output struct {
	Data CreateOrgV1OutputData

	http.Response
}

type CreateOrgV1OutputData struct {
	Id   string `json:"id"`
	Code string `json:"code"`
}

type CreateOrgV1Input struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

func (c Client) CreateOrgV1(input CreateOrgV1Input) (*CreateOrgV1Output, error) {
	var outputData CreateOrgV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPost,
		Path:   "/api/v1/org",
		Data:   input,
		Output: &outputData,
	})
	if err != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorOrgExists.Error():
			err = ErrorOrgExists
		}
	}
	return &CreateOrgV1Output{
		Data:     outputData,
		Response: outputClient.Response,
	}, err
}

type ListOrgsV1Output struct {
	Data ListOrgsV1OutputData

	http.Response
}

type ListOrgsV1OutputData []ListOrgsV1OutputDataOrg

type ListOrgsV1OutputDataOrg struct {
	Code       string     `json:"code"`
	CreatedAt  time.Time  `json:"createdAt"`
	Id         string     `json:"id"`
	JoinedAt   time.Time  `json:"joinedAt"`
	MemberType string     `json:"memberType"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	UpdatedAt  *time.Time `json:"updatedAt"`
}

func (c Client) ListOrgsV1() (*ListOrgsV1Output, error) {
	var outputData ListOrgsV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   "/api/v1/orgs",
		Output: &outputData,
	})
	return &ListOrgsV1Output{
		Data:     outputData,
		Response: outputClient.Response,
	}, err
}

type GetOrgV1Output struct {
	Data GetOrgV1OutputData
	http.Response
}

type GetOrgV1OutputData struct {
	Id         string     `json:"id"`
	Code       string     `json:"code"`
	Type       string     `json:"type"`
	Logo       *string    `json:"logo"`
	Icon       *string    `json:"icon"`
	Motd       *string    `json:"motd"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  *time.Time `json:"updatedAt"`
	IsDeleted  bool       `json:"isDeleted"`
	DeletedAt  *time.Time `json:"deletedAt"`
	IsDisabled bool       `json:"isDisabled"`
	DisabledAt *time.Time `json:"disabledAt"`
	UserCount  int        `json:"userCount"`
}

// GetOrgV1 retrieves the current organisation
func (c Client) GetOrgV1() (*GetOrgV1Output, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = "/api/v1/org"
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
	output := GetOrgV1Output{Response: *httpResponse}
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
	var data GetOrgV1OutputData
	if err := json.Unmarshal(responseData, &data); err != nil {
		return &output, fmt.Errorf("failed to unmarshal response data into output: %s", err)
	}
	output.Data = data
	return &output, nil
}
