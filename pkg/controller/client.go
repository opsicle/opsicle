package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"opsicle/internal/common"
	"opsicle/internal/types"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"syscall"
	"time"
)

var (
	DefaultClientTimeout = 10 * time.Second
)

type NewClientOpts struct {
	ControllerUrl  string
	BasicAuth      *NewClientBasicAuthOpts
	BearerAuth     *NewClientBearerAuthOpts
	Id             string
	RequestTimeout time.Duration
}

type NewClientBasicAuthOpts struct {
	Username string
	Password string
}

type NewClientBearerAuthOpts struct {
	Token string
}

func NewClient(opts NewClientOpts) (*Client, error) {
	var hostname, username string
	if host, err := os.Hostname(); err == nil {
		hostname = host
	} else {
		hostname = "unknown-host"
	}
	if user, _ := user.Current(); user != nil {
		username = user.Username
	} else {
		username = "unknown-user"
	}
	timeout := DefaultClientTimeout
	if opts.RequestTimeout != 0 {
		timeout = opts.RequestTimeout
	}
	client := &Client{
		BasicAuth:  opts.BasicAuth,
		BearerAuth: opts.BearerAuth,
		HttpClient: &http.Client{
			Timeout: timeout,
		},
		Id: filepath.Join(opts.Id, fmt.Sprintf("%s@%s", username, hostname)),
	}

	controllerUrl, err := url.Parse(opts.ControllerUrl)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse provided controllerUrl[%s]: %w", types.ErrorInvalidInput, opts.ControllerUrl, err)
	}

	if controllerUrl.Scheme == "" {
		return nil, fmt.Errorf("%w: failed to determine url scheme of controllerUrl[%s]", types.ErrorInvalidInput, opts.ControllerUrl)
	}
	client.ControllerUrl = controllerUrl

	healthcheckOutput, err := client.HealthcheckPing()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to check health of controller: %w", types.ErrorHealthcheckFailed, err)
	}
	if healthcheckOutput.Data.Status != "ok" {
		return nil, fmt.Errorf("%w: controller repsonded with unhealthy: %w", types.ErrorHealthcheckFailed, err)
	}

	return client, nil
}

type request struct {
	Method string
	Path   string
	Data   any
	Output any
}

func (c request) Validate() error {
	errs := []error{}
	if c.Output == nil {
		errs = append(errs, types.ErrorOutputNil)
	} else if reflect.TypeOf(c.Output).Kind() != reflect.Ptr {
		errs = append(errs, types.ErrorOutputNotPointer)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

type clientOutput struct {
	code    error
	message string

	http.Response
}

func (c clientOutput) Error() error {
	if c.code == nil {
		return nil
	}
	return fmt.Errorf("%w: %s", c.code, c.message)
}

func (c clientOutput) GetErrorCode() error {
	switch c.code.Error() {
	case types.ErrorInvalidCredentials.Error():
		return types.ErrorInvalidCredentials
	case types.ErrorInvalidInput.Error():
		return types.ErrorInvalidInput
	case types.ErrorInsufficientPermissions.Error():
		return types.ErrorInsufficientPermissions
	case types.ErrorDatabaseIssue.Error():
		return types.ErrorDatabaseIssue
	}
	return c.code
}

func (c clientOutput) GetMessage() string {
	return c.message
}

func (c clientOutput) GetStatusCode() int {
	return c.Response.StatusCode
}

func (c clientOutput) GetResponse() http.Response {
	return c.Response
}

type Client struct {
	// ControllerUrl is the URL where the approver service is accessible
	// at
	ControllerUrl *url.URL
	BasicAuth     *NewClientBasicAuthOpts
	BearerAuth    *NewClientBearerAuthOpts

	// HttpClient is the HTTP client
	HttpClient *http.Client

	// Id will be included in the user-agent for identification
	Id string
}

func (c Client) WithAuth(auth ...string) Client {
	if c.BearerAuth == nil {
		c.BearerAuth = &NewClientBearerAuthOpts{}
	}
	if len(auth) == 1 {
		c.BearerAuth.Token = auth[0]
	} else if len(auth) == 2 {
		c.BasicAuth.Username = auth[0]
		c.BasicAuth.Password = auth[1]
	}
	return c
}

func (c Client) addRequiredHeaders(httpRequest *http.Request) {
	httpRequest.Header.Add("Content-Type", "application/json")
	httpRequest.Header.Add("User-Agent", fmt.Sprintf("opsicle/controller-sdk/client-%s", c.Id))
	if c.BasicAuth != nil {
		httpRequest.SetBasicAuth(c.BasicAuth.Username, c.BasicAuth.Password)
	}
	if c.BearerAuth != nil {
		httpRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.BearerAuth.Token))
	}
}

func (c Client) do(input request) (*clientOutput, error) {
	controllerUrl := *c.ControllerUrl
	controllerUrl.Path = input.Path
	var requestBody *bytes.Buffer = nil
	if input.Data != nil {
		inputData, err := json.Marshal(input.Data)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", types.ErrorClientMarshalInput, err)
		}
		requestBody = bytes.NewBuffer(inputData)
	}
	var httpRequest *http.Request
	var httpRequestError error
	if requestBody != nil {
		httpRequest, httpRequestError = http.NewRequest(
			input.Method,
			controllerUrl.String(),
			requestBody,
		)
	} else {
		httpRequest, httpRequestError = http.NewRequest(
			input.Method,
			controllerUrl.String(),
			nil,
		)
	}
	if httpRequestError != nil {
		return nil, fmt.Errorf("%w: %w: %w", types.ErrorClientRequestCreation, types.ErrorOutputNil, httpRequestError)
	}
	c.addRequiredHeaders(httpRequest)
	httpResponse, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, fmt.Errorf("%w: %w: %w", types.ErrorConnectionRefused, types.ErrorOutputNil, err)
		} else if isTimeout(err) {
			return nil, fmt.Errorf("%w: %w: %w", types.ErrorConnectionTimedOut, types.ErrorOutputNil, err)
		}
		return nil, fmt.Errorf("%w: %w: %w", types.ErrorClientRequestExecution, types.ErrorOutputNil, err)
	}
	defer httpResponse.Body.Close()
	output := &clientOutput{Response: *httpResponse}
	if !isControllerResponse(httpResponse) {
		return output, types.ErrorClientResponseNotFromController
	}
	if httpResponse.StatusCode == http.StatusMethodNotAllowed {
		return output, types.ErrorInvalidEndpoint
	}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return output, fmt.Errorf("%w: %w", types.ErrorClientResponseReading, err)
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return output, fmt.Errorf("%w: %w", types.ErrorClientUnmarshalResponse, err)
	}
	output.message = response.Message
	output.code = errors.New(response.Code)
	if response.Data != nil && input.Output != nil {
		responseData, err := json.Marshal(response.Data)
		if err != nil {
			return output, fmt.Errorf("%w: %w", types.ErrorClientMarshalResponseData, err)
		}
		if err := json.Unmarshal(responseData, &input.Output); err != nil {
			return output, fmt.Errorf("%w: %w", types.ErrorClientUnmarshalOutput, err)
		}
	}
	if !response.Success {
		return output, fmt.Errorf("%w: received status code %v ('%s'): %w", types.ErrorClientUnsuccessfulResponse, output.GetStatusCode(), output.GetMessage(), output.GetErrorCode())
	}
	return output, nil
}

func isConnectionRefused(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED)
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
