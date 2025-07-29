package controller

import (
	"fmt"
	"net/http"
	"net/url"
)

type NewClientOpts struct {
	ControllerUrl string
	BasicAuth     *NewClientBasicAuthOpts
	BearerAuth    *NewClientBearerAuthOpts
	Id            string
}

type NewClientBasicAuthOpts struct {
	Username string
	Password string
}

type NewClientBearerAuthOpts struct {
	Token string
}

func NewClient(opts NewClientOpts) (*Client, error) {
	client := &Client{
		BasicAuth:  opts.BasicAuth,
		BearerAuth: opts.BearerAuth,
		HttpClient: &http.Client{},
		Id:         opts.Id,
	}

	controllerUrl, err := url.Parse(opts.ControllerUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse provided controllerUrl[%s]: %s", opts.ControllerUrl, err)
	}

	if controllerUrl.Scheme == "" {
		return nil, fmt.Errorf("failed to determine url scheme of controllerUrl[%s]", opts.ControllerUrl)
	}
	client.ControllerUrl = controllerUrl

	return client, nil
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
