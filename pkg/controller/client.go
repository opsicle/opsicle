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
		return nil, fmt.Errorf("failed to parse provided controllerUrl[%s]: %s", opts.ControllerUrl, err)
	}

	if controllerUrl.Scheme == "" {
		return nil, fmt.Errorf("failed to determine url scheme of controllerUrl[%s]", opts.ControllerUrl)
	}
	client.ControllerUrl = controllerUrl

	healthcheckOutput, err := client.HealthcheckPing()
	if err != nil || healthcheckOutput.Status != "success" {
		return nil, fmt.Errorf("failed to check health of controller: %w", err)
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
		errs = append(errs, ErrorOutputNil)
	} else if reflect.TypeOf(c.Output).Kind() != reflect.Ptr {
		errs = append(errs, ErrorOutputNotPointer)
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
	return c.code
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
			return nil, fmt.Errorf("%w: %w", ErrorClientMarshalInput, err)
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
		return nil, fmt.Errorf("%w: %w: %w", ErrorClientRequestCreation, ErrorOutputNil, httpRequestError)
	}
	c.addRequiredHeaders(httpRequest)
	httpResponse, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, fmt.Errorf("%w: %w: %w", ErrorConnectionRefused, ErrorOutputNil, err)
		} else if isTimeout(err) {
			return nil, fmt.Errorf("%w: %w: %w", ErrorConnectionTimedOut, ErrorOutputNil, err)
		}
		return nil, fmt.Errorf("%w: %w: %w", ErrorClientRequestExecution, ErrorOutputNil, err)
	}
	output := &clientOutput{Response: *httpResponse}
	if !isControllerResponse(httpResponse) {
		return output, ErrorClientResponseNotFromController
	}
	if httpResponse.StatusCode == http.StatusMethodNotAllowed {
		return output, ErrorInvalidEndpoint
	}
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return output, fmt.Errorf("%w: %w", ErrorClientResponseReading, err)
	}
	var response common.HttpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return output, fmt.Errorf("%w: %w", ErrorClientUnmarshalResponse, err)
	}
	output.message = response.Message
	output.code = errors.New(response.Code)
	if response.Data != nil {
		responseData, err := json.Marshal(response.Data)
		if err != nil {
			return output, fmt.Errorf("%w: %w", ErrorClientMarshalResponseData, err)
		}
		if err := json.Unmarshal(responseData, &input.Output); err != nil {
			return output, fmt.Errorf("%w: %w", ErrorClientUnmarshalOutput, err)
		}
	}
	if !response.Success {
		return output, fmt.Errorf("%w: received status code %v: %w", ErrorClientUnsuccessfulResponse, output.GetStatusCode(), output.GetErrorCode())
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
