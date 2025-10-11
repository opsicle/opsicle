package controller

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"opsicle/internal/controller"
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
	Type                  string `json:"type"`
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
	if err != nil && outputClient != nil {
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
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorOrgRequiresOneAdmin.Error():
			err = ErrorOrgRequiresOneAdmin
		case ErrorInsufficientPermissions.Error():
			err = ErrorInsufficientPermissions
		}
	}
	return output, err
}

type CanUserV1Output struct {
	Data CanUserV1OutputData `json:"data" yaml:"data"`

	http.Response
}

type CanUserV1OutputData controller.CanOrgUserActionV1OutputData

type CanUserV1Input struct {
	Action   string `json:"-"`
	OrgId    string `json:"-"`
	Resource string `json:"-"`
	UserId   string `json:"-"`
}

func (c Client) CanUserV1(opts CanUserV1Input) (*CanUserV1Output, error) {
	if opts.Action == "" {
		return nil, fmt.Errorf("action undefined: %w", ErrorInvalidInput)
	}
	if opts.OrgId == "" {
		return nil, fmt.Errorf("org undefined: %w", ErrorInvalidInput)
	}
	if opts.Resource == "" {
		return nil, fmt.Errorf("resource undefined: %w", ErrorInvalidInput)
	}
	if opts.UserId == "" {
		return nil, fmt.Errorf("user undefined: %w", ErrorInvalidInput)
	}
	var outputData CanUserV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/api/v1/org/%s/member/%s/can/%s/%s", opts.OrgId, opts.UserId, opts.Action, opts.Resource),
		Output: &outputData,
	})
	var output *CanUserV1Output
	if !errors.Is(err, ErrorOutputNil) && outputClient != nil {
		output = &CanUserV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	return output, err
}

type GetOrgV1Input struct {
	Ref string `json:"-"`
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
		Path:   fmt.Sprintf("/api/v1/org/%s", input.Ref),
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

type GetOrgMembershipV1Input struct {
	OrgId string `json:"-"`
}
type GetOrgMembershipV1Output struct {
	Data GetOrgMembershipV1OutputData
	http.Response
}

type GetOrgMembershipV1OutputData struct {
	JoinedAt   time.Time `json:"joinedAt"`
	MemberType string    `json:"memberType"`
	OrgCode    string    `json:"orgCode"`
	OrgId      string    `json:"orgId"`
	UserId     string    `json:"userId"`

	Permissions GetOrgMembershipV1OutputPermissions `json:"permissions"`
}

type GetOrgMembershipV1OutputPermissions struct {
	CanManageUsers bool `json:"canManageUsers"`
}

// GetOrgMembershipV1 retrieves the specified organisation using the org's codeword
func (c Client) GetOrgMembershipV1(input GetOrgMembershipV1Input) (*GetOrgMembershipV1Output, error) {
	var outputData GetOrgMembershipV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/api/v1/org/%s/member", input.OrgId),
		Output: &outputData,
	})
	var output *GetOrgMembershipV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &GetOrgMembershipV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	return output, err
}

type LeaveOrgV1Input struct {
	OrgId string `json:"-"`
}

type LeaveOrgV1Output struct {
	Data LeaveOrgV1OutputData

	http.Response
}

type LeaveOrgV1OutputData struct {
	IsSuccessful bool `json:"isSuccessful"`
}

func (c Client) LeaveOrgV1(input LeaveOrgV1Input) (*LeaveOrgV1Output, error) {
	var outputData LeaveOrgV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodDelete,
		Path:   fmt.Sprintf("/api/v1/org/%s/member", input.OrgId),
		Output: &outputData,
	})
	var output *LeaveOrgV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &LeaveOrgV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorOrgRequiresOneAdmin.Error():
			err = ErrorOrgRequiresOneAdmin
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
	Data ListOrgUsersV1OutputData `json:"data" yaml:"data"`
	http.Response
}

type ListOrgUsersV1OutputData []ListOrgUsersV1OutputDataUser

type ListOrgUsersV1OutputDataUser struct {
	JoinedAt   time.Time                          `json:"joinedAt" yaml:"joinedAt"`
	MemberType string                             `json:"memberType" yaml:"memberType"`
	OrgId      string                             `json:"orgId" yaml:"orgId"`
	OrgCode    string                             `json:"orgCode" yaml:"orgCode"`
	OrgName    string                             `json:"orgName" yaml:"orgName"`
	UserId     string                             `json:"userId" yaml:"userId"`
	UserEmail  string                             `json:"userEmail" yaml:"userEmail"`
	UserType   string                             `json:"userType" yaml:"userType"`
	Roles      []ListOrgUsersV1OutputDataUserRole `json:"roles" yaml:"roles"`
}

type ListOrgUsersV1OutputDataUserRole struct {
	CreatedAt     time.Time                                    `json:"createdAt" yaml:"createdAt"`
	CreatedBy     *ListOrgUsersV1OutputDataUserRoleUser        `json:"createdBy" yaml:"createdBy"`
	Id            string                                       `json:"id" yaml:"id"`
	LastUpdatedAt time.Time                                    `json:"lastUpdatedAt" yaml:"lastUpdatedAt"`
	Name          string                                       `json:"name" yaml:"name"`
	Permissions   []ListOrgUsersV1OutputDataUserRolePermission `json:"permissions" yaml:"permissions"`
}

type ListOrgUsersV1OutputDataUserRoleUser struct {
	Email string `json:"email" yaml:"email"`
	Id    string `json:"id" yaml:"id"`
}

type ListOrgUsersV1OutputDataUserRolePermission struct {
	Allows   uint64 `json:"allows" yaml:"allows"`
	Denys    uint64 `json:"denys" yaml:"denys"`
	Id       string `json:"id" yaml:"id"`
	Resource string `json:"resource" yaml:"resource"`
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

type ListOrgRolesV1Output struct {
	Data ListOrgRolesV1OutputData

	http.Response
}

type ListOrgRolesV1OutputData []ListOrgRolesV1OutputDataRole

type ListOrgRolesV1OutputDataRole struct {
	CreatedAt     time.Time                                `json:"createdAt" yaml:"createdAt"`
	CreatedBy     *ListOrgRolesV1OutputDataRoleUser        `json:"createdBy" yaml:"createdBy"`
	Id            string                                   `json:"id" yaml:"id"`
	LastUpdatedAt time.Time                                `json:"lastUpdatedAt" yaml:"lastUpdatedAt"`
	Name          string                                   `json:"name" yaml:"name"`
	OrgId         string                                   `json:"orgId" yaml:"orgId"`
	Permissions   []ListOrgRolesV1OutputDataRolePermission `json:"permissions" yaml:"permissions"`
}

type ListOrgRolesV1OutputDataRoleUser struct {
	Email string `json:"email" yaml:"email"`
	Id    string `json:"id" yaml:"id"`
}

type ListOrgRolesV1OutputDataRolePermission struct {
	Allows   uint64 `json:"allows" yaml:"allows"`
	Denys    uint64 `json:"denys" yaml:"denys"`
	Id       string `json:"id" yaml:"id"`
	Resource string `json:"resource" yaml:"resource"`
}

type ListOrgRolesV1Input struct {
	OrgId string `json:"-"`
}

func (c Client) ListOrgRolesV1(input ListOrgRolesV1Input) (*ListOrgRolesV1Output, error) {
	var outputData ListOrgRolesV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/api/v1/org/%s/roles", input.OrgId),
		Output: &outputData,
	})
	var output *ListOrgRolesV1Output
	if outputClient != nil {
		output = &ListOrgRolesV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	return output, err
}

type ListOrgTokensV1Output struct {
	Data controller.ListOrgTokensV1Output

	http.Response
}

type ListOrgTokensV1OutputData controller.ListOrgTokensV1Output

type ListOrgTokensV1Input struct {
	OrgId string `json:"-"`
}

func (c Client) ListOrgTokensV1(input ListOrgTokensV1Input) (*ListOrgTokensV1Output, error) {
	var outputData controller.ListOrgTokensV1Output
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/api/v1/org/%s/tokens", input.OrgId),
		Output: &outputData,
	})
	var output *ListOrgTokensV1Output
	if outputClient != nil {
		output = &ListOrgTokensV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorInsufficientPermissions.Error():
			err = ErrorInsufficientPermissions
		case ErrorNotFound.Error():
			err = ErrorNotFound
		}
	}
	return output, err
}

type GetOrgTokenV1Output struct {
	Data GetOrgTokenV1OutputData

	http.Response
}

type GetOrgTokenV1OutputData controller.GetOrgTokenV1Output

type GetOrgTokenV1Input struct {
	OrgId   string `json:"-"`
	TokenId string `json:"-"`
}

func (c Client) GetOrgTokenV1(input GetOrgTokenV1Input) (*GetOrgTokenV1Output, error) {
	var outputData GetOrgTokenV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/api/v1/org/%s/token/%s", input.OrgId, input.TokenId),
		Output: &outputData,
	})
	var output *GetOrgTokenV1Output
	if outputClient != nil {
		output = &GetOrgTokenV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorInsufficientPermissions.Error():
			err = ErrorInsufficientPermissions
		case ErrorNotFound.Error():
			err = ErrorNotFound
		}
	}
	return output, err
}

type CreateOrgTokenV1Input struct {
	OrgId       string  `json:"-"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	RoleId      string  `json:"roleId"`
}

type CreateOrgTokenV1Output struct {
	Data CreateOrgTokenV1OutputData

	http.Response
}

type CreateOrgTokenV1OutputData struct {
	TokenId        string `json:"tokenId" yaml:"tokenId"`
	Name           string `json:"name" yaml:"name"`
	ApiKey         string `json:"apiKey" yaml:"apiKey"`
	CertificatePem string `json:"certificatePem" yaml:"certificatePem"`
	PrivateKeyPem  string `json:"privateKeyPem" yaml:"privateKeyPem"`
}

func (c Client) CreateOrgTokenV1(input CreateOrgTokenV1Input) (*CreateOrgTokenV1Output, error) {
	var outputData CreateOrgTokenV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/org/%s/token", input.OrgId),
		Data:   input,
		Output: &outputData,
	})
	var output *CreateOrgTokenV1Output
	if outputClient != nil {
		output = &CreateOrgTokenV1Output{
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
