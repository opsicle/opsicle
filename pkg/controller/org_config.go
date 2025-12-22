package controller

import (
	"errors"
	"fmt"
	"net/http"
	"opsicle/internal/types"
)

type SetOrgApprovalsConfigV1Input struct {
	OrgId              string `json:"-" yaml:"-"`
	IsApprovalsEnabled bool   `json:"isApprovalsEnabled" yaml:"isApprovalsEnabled"`
}

type SetOrgApprovalsConfigV1OutputData struct {
	IsApprovalsEnabled bool `json:"isApprovalsEnabled" yaml:"isApprovalsEnabled"`
}

type SetOrgApprovalsConfigV1Output struct {
	Data SetOrgApprovalsConfigV1OutputData `json:"data" yaml:"data"`

	http.Response
}

func (c Client) SetOrgApprovalsConfigV1(input SetOrgApprovalsConfigV1Input) (*SetOrgApprovalsConfigV1Output, error) {
	if input.OrgId == "" {
		return nil, fmt.Errorf("org undefined: %w", types.ErrorInvalidInput)
	}

	var outputData SetOrgApprovalsConfigV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("/api/v1/org/%s/config/approvals", input.OrgId),
		Data:   input,
		Output: &outputData,
	})

	var output *SetOrgApprovalsConfigV1Output
	if !errors.Is(err, types.ErrorOutputNil) && outputClient != nil {
		output = &SetOrgApprovalsConfigV1Output{
			Data:     outputData,
			Response: outputClient.Response,
		}
	}

	if err != nil && outputClient != nil {
		switch outputClient.GetErrorCode().Error() {
		case types.ErrorInvalidInput.Error():
			err = types.ErrorInvalidInput
		case types.ErrorInsufficientPermissions.Error():
			err = types.ErrorInsufficientPermissions
		case types.ErrorNotFound.Error():
			err = types.ErrorNotFound
		}
	}

	return output, err
}
