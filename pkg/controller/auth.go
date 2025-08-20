package controller

import (
	"net/http"
)

type ResetPasswordV1Input struct {
	CurrentPassword  *string `json:"currentPassword,omitempty"`
	Email            *string `json:"email,omitempty"`
	NewPassword      *string `json:"newPassword,omitempty"`
	VerificationCode *string `json:"verificationCode,omitempty"`
}

type ResetPasswordV1Output struct {
	Data ResetPasswordV1OutputData

	http.Response
}

type ResetPasswordV1OutputData struct {
	IsSuccessful bool `json:"isSuccessful"`
}

func (c Client) ResetPasswordV1(opts ResetPasswordV1Input) (*ResetPasswordV1Output, error) {
	var outputData ResetPasswordV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPatch,
		Path:   "/api/v1/user/password",
		Data:   opts,
		Output: &outputData,
	})
	if err != nil {
		switch outputClient.GetErrorCode().Error() {
		case ErrorInvalidInput.Error():
			err = ErrorInvalidInput
		case ErrorInvalidCredentials.Error():
			err = ErrorInvalidCredentials
		}
	}
	return &ResetPasswordV1Output{
		Data:     outputData,
		Response: outputClient.GetResponse(),
	}, err
}
