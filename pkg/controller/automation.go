package controller

import (
	"errors"
	"fmt"
	"net/http"
	"opsicle/internal/controller"
)

type CreateAutomationV1Output struct {
	Data controller.CreateAutomationV1OutputData
	http.Response
}

type CreateAutomationV1Input controller.CreateAutomationV1Input

func (c Client) CreateAutomationV1(input CreateAutomationV1Input) (*CreateAutomationV1Output, error) {
	var outputData controller.CreateAutomationV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPost,
		Path:   "/api/v1/automation",
		Data:   input,
		Output: &outputData,
	})
	var output *CreateAutomationV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &CreateAutomationV1Output{
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

type RunAutomationV1Output struct {
	Data controller.RunAutomationV1Output
	http.Response
}

type RunAutomationV1Input controller.RunAutomationV1Input

func (c Client) RunAutomationV1(input RunAutomationV1Input) (*RunAutomationV1Output, error) {
	var outputData controller.RunAutomationV1Output
	outputClient, err := c.do(request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/automation/%s", input.AutomationId),
		Data:   input,
		Output: &outputData,
	})
	var output *RunAutomationV1Output = nil
	if !errors.Is(err, ErrorOutputNil) {
		output = &RunAutomationV1Output{
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
