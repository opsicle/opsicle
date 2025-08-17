package controller

import (
	"fmt"
	"net/http"
	"opsicle/internal/controller/models"
)

const (
	MfaTypeTotp = "totp"
)

type ListUserMfasV1Input struct{}

type ListUserMfasV1Output struct {
	Data ListUserMfasV1OutputData

	http.Response
}

type ListUserMfasV1OutputData []models.UserMfa

func (c Client) ListUserMfasV1(opts ListUserMfasV1Input) (*ListUserMfasV1Output, error) {
	var outputData ListUserMfasV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   "/api/v1/user/mfas",
		Output: &outputData,
	})
	return &ListUserMfasV1Output{
		Data:     outputData,
		Response: outputClient.GetResponse(),
	}, err
}

type ListAvailableMfaTypesOutput struct {
	Data []ListAvailableMfaTypesOutputType `json:"data"`

	http.Response
}

type ListAvailableMfaTypesOutputType struct {
	Description string `json:"description"`
	Label       string `json:"label"`
	Value       string `json:"value"`
}

func (c Client) ListAvailableMfaTypes() (*ListAvailableMfaTypesOutput, error) {
	var outputData []ListAvailableMfaTypesOutputType
	outputClient, err := c.do(request{
		Method: http.MethodOptions,
		Path:   "/api/v1/user/mfas",
		Output: &outputData,
	})
	return &ListAvailableMfaTypesOutput{
		Data:     outputData,
		Response: outputClient.Response,
	}, err
}

type CreateUserMfaV1Input struct {
	Password string `json:"password"`
	MfaType  string `json:"mfaType"`
}

type CreateUserMfaV1Output struct {
	Data CreateUserMfaV1OutputData

	http.Response
}

type CreateUserMfaV1OutputData struct {
	Id        string `json:"id"`
	Secret    string `json:"secret"`
	Type      string `json:"type"`
	UserEmail string `json:"userEmail"`
	UserId    string `json:"userId"`
}

func (c Client) CreateUserMfaV1(input CreateUserMfaV1Input) (*CreateUserMfaV1Output, error) {
	var outputData CreateUserMfaV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPost,
		Path:   "/api/v1/user/mfa",
		Data:   input,
		Output: &outputData,
	})
	if err != nil {
		if outputClient.GetErrorCode().Error() == ErrorInvalidCredentials.Error() {
			err = fmt.Errorf("password verification failed: %w", ErrorInvalidCredentials)
		}
	}
	return &CreateUserMfaV1Output{
		Data:     outputData,
		Response: outputClient.GetResponse(),
	}, err
}

type VerifyUserMfaV1Input struct {
	Id    string `json:"-"`
	Value string `json:"value"`
}

type VerifyUserMfaV1Output struct {
	Data VerifyUserMfaV1OutputData

	http.Response
}

type VerifyUserMfaV1OutputData struct {
	Id     string `json:"id"`
	Type   string `json:"type"`
	UserId string `json:"userId"`
}

func (c Client) VerifyUserMfaV1(input VerifyUserMfaV1Input) (*VerifyUserMfaV1Output, error) {
	var outputData VerifyUserMfaV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/user/mfa/%v", input.Id),
		Data:   input,
		Output: &outputData,
	})
	return &VerifyUserMfaV1Output{
		Data:     outputData,
		Response: outputClient.GetResponse(),
	}, err
}
