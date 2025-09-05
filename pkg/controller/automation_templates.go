package controller

import (
	"errors"
	"net/http"
)

type SubmitAutomationTemplateV1Output struct {
	Data SubmitAutomationTemplateV1OutputData

	http.Response
}

type SubmitAutomationTemplateV1OutputData struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Version int64  `json:"version"`
}

type SubmitAutomationTemplateV1Input struct {
	Data []byte `json:"data"`
}

func (c Client) SubmitAutomationTemplateV1(input SubmitAutomationTemplateV1Input) (*SubmitAutomationTemplateV1Output, error) {
	var outputData SubmitAutomationTemplateV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPost,
		Path:   "/api/v1/automation-templates",
		Data:   input,
		Output: &outputData,
	})
	var output *SubmitAutomationTemplateV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &SubmitAutomationTemplateV1Output{
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
