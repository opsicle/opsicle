package controller

import (
	"errors"
	"net/http"
	"opsicle/internal/types"
)

type HealthcheckPingOutput struct {
	Data HealthcheckPingOutputData

	http.Response
}

type HealthcheckPingOutputData struct {
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Status   string   `json:"status"`
}

func (c Client) HealthcheckPing() (*HealthcheckPingOutput, error) {
	var outputData HealthcheckPingOutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   "/healthz",
		Output: &outputData,
	})
	if err != nil {
		if errors.Is(err, types.ErrorOutputNil) {
			return nil, err
		}
	}
	return &HealthcheckPingOutput{
		Data:     outputData,
		Response: outputClient.Response,
	}, err
}
