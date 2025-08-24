package controller

import (
	"errors"
	"fmt"
	"net/http"
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
	var output *CreateOrgV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &CreateOrgV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorOrgExists.Error():
			err = ErrorOrgExists
		}
	}
	return output, err
}

type CreateOrgUserV1Input struct {
	Email                 string `json:"email"`
	OrgId                 string `json:"-"`
	IsTriggerEmailEnabled bool   `json:"isTriggerEmailEnabled"`
}

type CreateOrgUserV1Output struct {
	Data CreateOrgUserV1OutputData

	http.Response
}

type CreateOrgUserV1OutputData struct {
	Id             string `json:"id"`
	JoinCode       string `json:"joinCode"`
	IsExistingUser bool   `json:"isExistingUser"`
}

func (c Client) CreateOrgUserV1(input CreateOrgUserV1Input) (*CreateOrgUserV1Output, error) {
	var outputData CreateOrgUserV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/org/%s/member", input.OrgId),
		Data:   input,
		Output: &outputData,
	})
	var output *CreateOrgUserV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &CreateOrgUserV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorInvitationExists.Error():
			err = ErrorInvitationExists
		case ErrorUserExistsInOrg.Error():
			err = ErrorUserExistsInOrg
		}
	}
	return output, err
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
	var output *ListOrgsV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &ListOrgsV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	return output, err
}

type ListOrgInvitationsV1Output struct {
	Data ListOrgInvitationsV1OutputData

	http.Response
}

type ListOrgInvitationsV1OutputData struct {
	Invitations []ListOrgInvitationsV1OutputDataOrg `json:"invitations"`
}

type ListOrgInvitationsV1OutputDataOrg struct {
	Id           string    `json:"id"`
	InvitedAt    time.Time `json:"invitedAt"`
	InviterId    string    `json:"inviterId"`
	InviterEmail string    `json:"inviterEmail"`
	JoinCode     string    `json:"joinCode"`
	OrgCode      string    `json:"orgCode"`
	OrgName      string    `json:"orgName"`
}

func (c Client) ListOrgInvitationsV1() (*ListOrgInvitationsV1Output, error) {
	var outputData ListOrgInvitationsV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   "/api/v1/user/org-invitations",
		Output: &outputData,
	})
	var output *ListOrgInvitationsV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &ListOrgInvitationsV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	return output, err
}

type GetOrgV1Input struct {
	Code string `json:"code"`
}
type GetOrgV1Output struct {
	Data GetOrgV1OutputData
	http.Response
}

type GetOrgV1OutputData struct {
	Code      string     `json:"code"`
	CreatedAt time.Time  `json:"createdAt"`
	Id        string     `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

// GetOrgV1 retrieves the specified organisation using the org's codeword
func (c Client) GetOrgV1(input GetOrgV1Input) (*GetOrgV1Output, error) {
	var outputData GetOrgV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/api/v1/org/%s", input.Code),
		Output: &outputData,
	})
	var output *GetOrgV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &GetOrgV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	return output, err
}
