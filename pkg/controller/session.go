package controller

import (
	"fmt"
	"net/http"
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
		case ErrorEmailUnverified.Error():
			err = ErrorEmailUnverified
		case ErrorInvalidCredentials.Error():
			err = ErrorInvalidCredentials
		case ErrorMfaRequired.Error():
			err = ErrorMfaRequired
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
		case ErrorMfaTokenInvalid.Error():
			err = ErrorMfaTokenInvalid
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
		case err.Error() == ErrorJwtTokenSignature.Error():
			return nil, ErrorJwtTokenSignature
		case err.Error() == ErrorJwtClaimsInvalid.Error():
			return nil, ErrorJwtClaimsInvalid
		case err.Error() == ErrorJwtTokenExpired.Error():
			return nil, ErrorJwtTokenExpired
		case err.Error() == ErrorUnknown.Error():
			return nil, ErrorUnknown
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
	IsExpired time.Time `json:"isExpired"`
	ExpiresAt time.Time `json:"expiresAt"`
	StartedAt time.Time `json:"startedAt"`
	UserId    string    `json:"userId"`
	Username  string    `json:"username"`
	UserType  string    `json:"userType"`
	OrgCode   *string   `json:"orgCode"`
	OrgId     *string   `json:"orgId"`
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
		case ErrorAuthRequired.Error():
			err = ErrorAuthRequired
		case ErrorSessionExpired.Error():
			err = ErrorSessionExpired
		}
	}
	return &ValidateSessionV1Output{
		Data:     outputData,
		Response: outputClient.GetResponse(),
	}, err

	// output.Data.IsExpired = output.Data.ExpiresAt.Before(time.Now())
}
