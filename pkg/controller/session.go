package controller

import (
	"fmt"
	"net/http"
	"opsicle/internal/types"
	"time"
)

type CreateSessionV1Input struct {
	OrgCode  *string `json:"orgCode"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Hostname string  `json:"hostname"`
}

type CreateSessionV1Output struct {
	Data CreateSessionV1OutputData

	http.Response
}

type CreateSessionV1OutputData struct {
	SessionId    string `json:"sessionId"`
	SessionToken string `json:"sessionToken"`

	// MfaType is only populated if `ErrorMfaRequired` is returned
	// in the `error` return
	MfaType *string `json:"mfaType"`

	// LoginId is only populated if `ErrorMfaRequired` is returned
	// in the `error` return
	LoginId *string `json:"loginId"`
}

type createSessionV1MfaRequiredResponse struct {
	LoginId string `json:"loginId"`
	MfaType string `json:"mfaType"`
}

func (c Client) CreateSessionV1(opts CreateSessionV1Input) (*CreateSessionV1Output, error) {
	var outputData CreateSessionV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPost,
		Path:   "/api/v1/session",
		Data:   opts,
		Output: &outputData,
	})
	if err != nil {
		if outputClient == nil {
			return nil, err
		}
		switch outputClient.GetErrorCode().Error() {
		case types.ErrorNotFound.Error():
			err = types.ErrorNotFound
		case types.ErrorEmailUnverified.Error():
			err = types.ErrorEmailUnverified
		case types.ErrorInvalidCredentials.Error():
			err = types.ErrorInvalidCredentials
		case types.ErrorMfaRequired.Error():
			err = types.ErrorMfaRequired
		}
	}
	return &CreateSessionV1Output{
		Data:     outputData,
		Response: outputClient.GetResponse(),
	}, err
}

type StartSessionWithMfaV1Input struct {
	Hostname string `json:"hostname"`
	LoginId  string `json:"-"`
	MfaType  string `json:"mfaType"`
	MfaToken string `json:"mfaToken"`
}

func (c Client) StartSessionWithMfaV1(opts StartSessionWithMfaV1Input) (*CreateSessionV1Output, error) {
	var outputData CreateSessionV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/session/mfa/%s", opts.LoginId),
		Data:   opts,
		Output: &outputData,
	})
	if err != nil {
		switch outputClient.GetErrorCode().Error() {
		case types.ErrorMfaTokenInvalid.Error():
			err = types.ErrorMfaTokenInvalid
		}
	}
	return &CreateSessionV1Output{
		Data:     outputData,
		Response: outputClient.GetResponse(),
	}, err
}

type DeleteSessionV1Output struct {
	Data DeleteSessionV1OutputData

	http.Response
}

type DeleteSessionV1OutputData struct {
	// SessionId is only populated if the call to the controller was
	// successful as indicated by the `.IsSuccessful` property
	SessionId string `json:"sessionId"`
}

func (c Client) DeleteSessionV1() (*DeleteSessionV1Output, error) {
	var outputData DeleteSessionV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodDelete,
		Path:   "/api/v1/session",
		Output: &outputData,
	})
	if err != nil {
		switch true {
		case err.Error() == types.ErrorJwtTokenSignature.Error():
			return nil, types.ErrorJwtTokenSignature
		case err.Error() == types.ErrorJwtClaimsInvalid.Error():
			return nil, types.ErrorJwtClaimsInvalid
		case err.Error() == types.ErrorJwtTokenExpired.Error():
			return nil, types.ErrorJwtTokenExpired
		case err.Error() == types.ErrorUnknown.Error():
			return nil, types.ErrorUnknown
		}
	}
	return &DeleteSessionV1Output{
		Data:     outputData,
		Response: outputClient.GetResponse(),
	}, err
}

type ValidateSessionV1Output struct {
	Data ValidateSessionV1OutputData

	http.Response
}

type ValidateSessionV1OutputData struct {
	ExpiresAt time.Time `json:"expiresAt"`
	Id        string    `json:"id"`
	IsExpired bool      `json:"isExpired"`
	UserId    string    `json:"userId"`
}

func (c Client) ValidateSessionV1() (*ValidateSessionV1Output, error) {
	var outputData ValidateSessionV1OutputData
	outputClient, err := c.do(request{
		Method: http.MethodGet,
		Path:   "/api/v1/session",
		Output: &outputData,
	})
	if err != nil {
		switch outputClient.GetErrorCode().Error() {
		case types.ErrorAuthRequired.Error():
			err = types.ErrorAuthRequired
		case types.ErrorSessionExpired.Error():
			err = types.ErrorSessionExpired
		}
	}
	return &ValidateSessionV1Output{
		Data:     outputData,
		Response: outputClient.GetResponse(),
	}, err

	// output.Data.IsExpired = output.Data.ExpiresAt.Before(time.Now())
}
