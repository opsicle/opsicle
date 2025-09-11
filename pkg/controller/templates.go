package controller

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

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
