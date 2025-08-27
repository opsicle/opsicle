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

type DeleteOrgUserV1Input struct {
	OrgId  string `json:"-"`
	UserId string `json:"-"`
}

type DeleteOrgUserV1Output struct {
	Data DeleteOrgUserV1OutputData

	http.Response
}

type DeleteOrgUserV1OutputData struct {
	IsSuccessful bool `json:"isSuccessful"`
}

func (c Client) DeleteOrgUserV1(input DeleteOrgUserV1Input) (*DeleteOrgUserV1Output, error) {
	var outputData DeleteOrgUserV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodDelete,
		Path:   fmt.Sprintf("/api/v1/org/%s/member/%s", input.OrgId, input.UserId),
		Output: &outputData,
	})
	var output *DeleteOrgUserV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &DeleteOrgUserV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorInvalidInput.Error():
			err = fmt.Errorf("%s: %w", outputClient.GetMessage(), ErrorInvalidInput)
		case ErrorInsufficientPermissions.Error():
			err = ErrorInsufficientPermissions
		case ErrorDatabaseIssue.Error():
			err = ErrorDatabaseIssue
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
	OrgId        string    `json:"orgId"`
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

type ListOrgUsersV1Output struct {
	Data ListOrgUsersV1OutputData
	http.Response
}

type ListOrgUsersV1OutputData []ListOrgUsersV1OutputDataUser

type ListOrgUsersV1OutputDataUser struct {
	JoinedAt   time.Time `json:"joinedAt"`
	MemberType string    `json:"memberType"`
	OrgId      string    `json:"orgId"`
	OrgCode    string    `json:"orgCode"`
	OrgName    string    `json:"orgName"`
	UserId     string    `json:"userId"`
	UserEmail  string    `json:"userEmail"`
	UserType   string    `json:"userType"`
}

type ListOrgUsersV1Input struct {
	OrgId string `json:"-"`
}

func (c Client) ListOrgUsersV1(input ListOrgUsersV1Input) (*ListOrgUsersV1Output, error) {
	var outputData ListOrgUsersV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/api/v1/org/%s/members", input.OrgId),
		Output: &outputData,
	})
	var output *ListOrgUsersV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &ListOrgUsersV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	return output, err
}

type ListOrgMemberTypesV1Output struct {
	Data ListOrgMemberTypesV1OutputData

	http.Response
}

type ListOrgMemberTypesV1OutputData []string

func (c Client) ListOrgMemberTypesV1() (*ListOrgMemberTypesV1Output, error) {
	var outputData ListOrgMemberTypesV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   "/api/v1/org/member/types",
		Output: &outputData,
	})
	var output *ListOrgMemberTypesV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &ListOrgMemberTypesV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	return output, err
}

type UpdateOrgInvitationV1Output struct {
	Data UpdateOrgInvitationV1OutputData

	http.Response
}

type UpdateOrgInvitationV1OutputData struct {
	JoinedAt       time.Time `json:"joinedAt"`
	MembershipType string    `json:"membershipType"`
	OrgId          string    `json:"orgId"`
	OrgCode        string    `json:"orgCode"`
	OrgName        string    `json:"orgName"`
	UserId         string    `json:"userId"`
}

type UpdateOrgInvitationV1Input struct {
	Id           string `json:"-"`
	IsAcceptance bool   `json:"isAcceptance"`
	JoinCode     string `json:"joinCode"`
}

func (c Client) UpdateOrgInvitationV1(input UpdateOrgInvitationV1Input) (*UpdateOrgInvitationV1Output, error) {
	var outputData UpdateOrgInvitationV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPatch,
		Path:   fmt.Sprintf("/api/v1/org/invitation/%s", input.Id),
		Data:   input,
		Output: &outputData,
	})
	var output *UpdateOrgInvitationV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &UpdateOrgInvitationV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	return output, err
}

type UpdateOrgUserV1Output struct {
	Data UpdateOrgUserV1OutputData

	http.Response
}

type UpdateOrgUserV1OutputData struct {
	IsSuccessful bool `json:"isSuccessful"`
}

type UpdateOrgUserV1Input struct {
	OrgId  string         `json:"-"`
	User   string         `json:"user"`
	Update map[string]any `json:"update"`
}

func (c Client) UpdateOrgUserV1(input UpdateOrgUserV1Input) (*UpdateOrgUserV1Output, error) {
	var outputData UpdateOrgUserV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPatch,
		Path:   fmt.Sprintf("/api/v1/org/%s/member", input.OrgId),
		Data:   input,
		Output: &outputData,
	})
	var output *UpdateOrgUserV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &UpdateOrgUserV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	return output, err
}
