package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
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
	client := &Client{
		BasicAuth:  opts.BasicAuth,
		BearerAuth: opts.BearerAuth,
		HttpClient: &http.Client{},
		Id:         filepath.Join(opts.Id, fmt.Sprintf("%s@%s", username, hostname)),
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
