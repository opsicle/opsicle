package controller

import (
	"errors"
	"net/http"
)

type CreateAutomationTemplateV1Output struct {
	Data CreateAutomationTemplateV1OutputData

	http.Response
}

type CreateAutomationTemplateV1OutputData struct {
	Id string `json:"id"`
}

type CreateAutomationTemplateV1Input struct {
	Data []byte `json:"data"`
}

func (c Client) CreateAutomationTemplateV1(input CreateAutomationTemplateV1Input) (*CreateAutomationTemplateV1Output, error) {
	var outputData CreateAutomationTemplateV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPost,
		Path:   "/api/v1/automation-templates",
		Data:   input,
		Output: &outputData,
	})
	var output *CreateAutomationTemplateV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &CreateAutomationTemplateV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}
	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorEmailExists.Error():
			err = ErrorEmailExists
		}
	}
	return output, err
}
