package controller

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type CreateTemplateUserV1Output struct {
	Data CreateTemplateUserV1OutputData
	http.Response
}

type CreateTemplateUserV1OutputData struct {
	Id             string `json:"id"`
	JoinCode       string `json:"joinCode"`
	IsExistingUser bool   `json:"isExistingUser"`
}

type CreateTemplateUserV1Input struct {
	TemplateId string  `json:"-"`
	UserId     *string `json:"userId"`
	UserEmail  *string `json:"userEmail"`
	CanView    bool    `json:"canView"`
	CanExecute bool    `json:"canExecute"`
	CanUpdate  bool    `json:"canUpdate"`
	CanDelete  bool    `json:"canDelete"`
	CanInvite  bool    `json:"canInvite"`
}

func (c Client) CreateTemplateUserV1(input CreateTemplateUserV1Input) (*CreateTemplateUserV1Output, error) {
	var outputData CreateTemplateUserV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/template/%s/user", input.TemplateId),
		Data:   input,
		Output: &outputData,
	})
	var output *CreateTemplateUserV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &CreateTemplateUserV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorDatabaseIssue.Error():
			err = ErrorDatabaseIssue
		case ErrorInsufficientPermissions.Error():
			err = ErrorInsufficientPermissions
		}
	}
	return output, err
}

type DeleteTemplateV1Output struct {
	Data DeleteTemplateV1OutputData
	http.Response
}

type DeleteTemplateV1OutputData struct {
	IsSuccessful bool   `json:"isSuccessful"`
	TemplateId   string `json:"templateId"`
	TemplateName string `json:"templateName"`
}

type DeleteTemplateV1Input struct {
	TemplateId string `json:"-"`
}

func (c *Client) DeleteTemplateV1(input DeleteTemplateV1Input) (*DeleteTemplateV1Output, error) {
	if _, err := uuid.Parse(input.TemplateId); err != nil {
		return nil, fmt.Errorf("%w: template id not a uuid", ErrorInvalidInput)
	}
	var outputData DeleteTemplateV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodDelete,
		Path:   fmt.Sprintf("/api/v1/template/%s", input.TemplateId),
		Output: &outputData,
	})
	var output *DeleteTemplateV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &DeleteTemplateV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorDatabaseIssue.Error():
			err = ErrorDatabaseIssue
		}
	}
	return output, err
}

type ListTemplatesV1Output struct {
	Data ListTemplatesV1OutputData
	http.Response
}

type ListTemplatesV1OutputData []ListTemplatesV1OutputDataTemplate

type ListTemplatesV1OutputDataTemplate struct {
	Id            string                                 `json:"id"`
	Content       string                                 `json:"content"`
	Description   string                                 `json:"description"`
	Name          string                                 `json:"name"`
	Version       int                                    `json:"version"`
	CreatedAt     time.Time                              `json:"createdAt"`
	CreatedBy     *ListTemplatesV1OutputDataTemplateUser `json:"createdBy"`
	LastUpdatedAt *time.Time                             `json:"lastUpdatedAt"`
	LastUpdatedBy *ListTemplatesV1OutputDataTemplateUser `json:"lastUpdatedBy"`
}

type ListTemplatesV1OutputDataTemplateUser struct {
	Id    string `json:"id"`
	Email string `json:"email"`
}

type ListTemplatesV1Input struct {
	Limit int `json:"limit"`
}

func (c Client) ListTemplatesV1(input ListTemplatesV1Input) (*ListTemplatesV1Output, error) {
	var outputData ListTemplatesV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   "/api/v1/templates",
		Data:   input,
		Output: &outputData,
	})
	var output *ListTemplatesV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &ListTemplatesV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorDatabaseIssue.Error():
			err = ErrorDatabaseIssue
		}
	}
	return output, err
}

type ListUserTemplateInvitationsV1Output struct {
	Data ListUserTemplateInvitationsV1OutputData
	http.Response
}

type ListUserTemplateInvitationsV1OutputData struct {
	Invitations []ListUserTemplateInvitationsV1OutputDataInvitation `json:"invitations"`
}

type ListUserTemplateInvitationsV1OutputDataInvitation struct {
	Id           string    `json:"id"`
	InvitedAt    time.Time `json:"invitedAt"`
	InviterId    string    `json:"inviterId"`
	InviterEmail string    `json:"inviterEmail"`
	JoinCode     string    `json:"joinCode"`
	TemplateId   string    `json:"templateId"`
	TemplateName string    `json:"templateName"`
}

func (c Client) ListUserTemplateInvitationsV1() (*ListUserTemplateInvitationsV1Output, error) {
	var outputData ListUserTemplateInvitationsV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   "/api/v1/user/template-invitations",
		Output: &outputData,
	})
	var output *ListUserTemplateInvitationsV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &ListUserTemplateInvitationsV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorDatabaseIssue.Error():
			err = ErrorDatabaseIssue
		}
	}
	return output, err
}

type ListTemplateUsersV1Output struct {
	Data ListTemplateUsersV1OutputData
	http.Response
}

type ListTemplateUsersV1OutputData struct {
	Users []ListTemplateUsersV1OutputDataUser `json:"users"`
}

type ListTemplateUsersV1OutputDataUser struct {
	Id         string    `json:"id"`
	Email      string    `json:"email"`
	CanView    bool      `json:"canView"`
	CanExecute bool      `json:"canExecute"`
	CanUpdate  bool      `json:"canUpdate"`
	CanDelete  bool      `json:"canDelete"`
	CanInvite  bool      `json:"canInvite"`
	CreatedAt  time.Time `json:"createdAt"`
}
type ListTemplateUsersV1Input struct {
	TemplateId string `json:"-"`
}

func (c Client) ListTemplateUsersV1(input ListTemplateUsersV1Input) (*ListTemplateUsersV1Output, error) {
	var outputData ListTemplateUsersV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/api/v1/template/%s/users", input.TemplateId),
		Output: &outputData,
	})
	var output *ListTemplateUsersV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &ListTemplateUsersV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorDatabaseIssue.Error():
			err = ErrorDatabaseIssue
		}
	}
	return output, err
}

type ListTemplateVersionsV1Output struct {
	Data ListTemplateVersionsV1OutputData
	http.Response
}

type ListTemplateVersionsV1OutputData struct {
	Template ListTemplateVersionsV1OutputTemplate  `json:"template"`
	Versions []ListTemplateVersionsV1OutputVersion `json:"versions"`
}

type ListTemplateVersionsV1OutputTemplate struct {
	Id            string                           `json:"id"`
	Name          string                           `json:"name"`
	Description   *string                          `json:"description"`
	Version       int64                            `json:"version"`
	CreatedAt     time.Time                        `json:"createdAt"`
	CreatedBy     ListTemplateVersionsV1OutputUser `json:"createdBy"`
	LastUpdatedAt time.Time                        `json:"lastUpdatedAt"`
	LastUpdatedBy ListTemplateVersionsV1OutputUser `json:"lastUpdatedBy"`
}

type ListTemplateVersionsV1OutputVersion struct {
	Content   string                           `json:"content"`
	CreatedAt time.Time                        `json:"createdAt"`
	CreatedBy ListTemplateVersionsV1OutputUser `json:"createdBy"`
	Version   int64                            `json:"version"`
}

type ListTemplateVersionsV1OutputUser struct {
	Id    string `json:"id"`
	Email string `json:"email"`
}

type ListTemplateVersionsV1Input struct {
	TemplateId string `json:"-"`
}

func (c Client) ListTemplateVersionsV1(input ListTemplateVersionsV1Input) (*ListTemplateVersionsV1Output, error) {
	var outputData ListTemplateVersionsV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/api/v1/template/%s/versions", input.TemplateId),
		Output: &outputData,
	})
	var output *ListTemplateVersionsV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &ListTemplateVersionsV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorDatabaseIssue.Error():
			err = ErrorDatabaseIssue
		}
	}
	return output, err
}

type DeleteTemplateUserV1Output struct {
	Data DeleteTemplateUserV1OutputData
	http.Response
}

type DeleteTemplateUserV1OutputData struct {
	IsSuccessful bool `json:"isSuccessful"`
}

type DeleteTemplateUserV1Input struct {
	TemplateId string `json:"-"`
	UserId     string `json:"-"`
}

func (c Client) DeleteTemplateUserV1(input DeleteTemplateUserV1Input) (*DeleteTemplateUserV1Output, error) {
	var outputData DeleteTemplateUserV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodDelete,
		Path:   fmt.Sprintf("/api/v1/template/%s/user/%s", input.TemplateId, input.UserId),
		Output: &outputData,
	})
	var output *DeleteTemplateUserV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &DeleteTemplateUserV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorLastUserInResource.Error():
			err = ErrorLastUserInResource
		case ErrorLastManagerOfResource.Error():
			err = ErrorLastManagerOfResource
		case ErrorDatabaseIssue.Error():
			err = ErrorDatabaseIssue
		case ErrorInsufficientPermissions.Error():
			err = ErrorInsufficientPermissions
		case ErrorNotFound.Error():
			err = ErrorNotFound
		}
	}
	return output, err
}

type SubmitTemplateV1Output struct {
	Data SubmitTemplateV1OutputData
	http.Response
}

type SubmitTemplateV1OutputData struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Version int64  `json:"version"`
}

type SubmitTemplateV1Input struct {
	Data []byte `json:"data"`
}

func (c Client) SubmitTemplateV1(input SubmitTemplateV1Input) (*SubmitTemplateV1Output, error) {
	var outputData SubmitTemplateV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPost,
		Path:   "/api/v1/template",
		Data:   input,
		Output: &outputData,
	})
	var output *SubmitTemplateV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &SubmitTemplateV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorDatabaseIssue.Error():
			err = ErrorDatabaseIssue
		}
	}
	return output, err
}

type UpdateTemplateDefaultVersionV1Output struct {
	Data UpdateTemplateDefaultVersionV1OutputData

	http.Response
}

type UpdateTemplateDefaultVersionV1OutputData struct {
	Version int64 `json:"version"`
}

type UpdateTemplateDefaultVersionV1Input struct {
	TemplateId string `json:"-"`
	Version    int64  `json:"version"`
}

func (c Client) UpdateTemplateDefaultVersionV1(input UpdateTemplateDefaultVersionV1Input) (*UpdateTemplateDefaultVersionV1Output, error) {
	var outputData UpdateTemplateDefaultVersionV1OutputData
	if _, err := uuid.Parse(input.TemplateId); err != nil {
		return nil, fmt.Errorf("invalid template id")
	}
	outputClient, err := c.do(request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("/api/v1/template/%s/version", input.TemplateId),
		Data:   input,
		Output: &outputData,
	})
	var output *UpdateTemplateDefaultVersionV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &UpdateTemplateDefaultVersionV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorDatabaseIssue.Error():
			err = ErrorDatabaseIssue
		case ErrorNotFound.Error():
			err = ErrorNotFound
		}
	}
	return output, err
}

type UpdateTemplateInvitationV1Output struct {
	Data UpdateTemplateInvitationV1OutputData
	http.Response
}

type UpdateTemplateInvitationV1OutputData struct {
	IsSuccessful bool                                      `json:"isSuccessful"`
	TemplateUser *UpdateTemplateInvitationV1OutputDataUser `json:"templateUser,omitempty"`
}

type UpdateTemplateInvitationV1OutputDataUser struct {
	UserId       string `json:"userId"`
	UserEmail    string `json:"userEmail"`
	TemplateId   string `json:"templateId"`
	TemplateName string `json:"templateName"`
	CanView      bool   `json:"canView"`
	CanExecute   bool   `json:"canExecute"`
	CanUpdate    bool   `json:"canUpdate"`
	CanDelete    bool   `json:"canDelete"`
	CanInvite    bool   `json:"canInvite"`
	CreatedBy    string `json:"createdBy"`
}

type UpdateTemplateInvitationV1Input struct {
	Id           string `json:"-"`
	IsAcceptance bool   `json:"isAcceptance"`
	JoinCode     string `json:"joinCode"`
}

func (c Client) UpdateTemplateInvitationV1(input UpdateTemplateInvitationV1Input) (*UpdateTemplateInvitationV1Output, error) {
	var outputData UpdateTemplateInvitationV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPatch,
		Path:   fmt.Sprintf("/api/v1/template/invitation/%s", input.Id),
		Data:   input,
		Output: &outputData,
	})
	var output *UpdateTemplateInvitationV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &UpdateTemplateInvitationV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorDatabaseIssue.Error():
			err = ErrorDatabaseIssue
		case ErrorNotFound.Error():
			err = ErrorNotFound
		}
	}
	return output, err
}
