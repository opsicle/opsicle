package controller

import (
	"fmt"
	"net/url"
	"opsicle/internal/common"
)

const (
	sessionCachePrefix = "session"
)

var publicServerUrl string

func SetPublicServerUrl(publicUrl string) error {
	urlInstance, err := url.Parse(publicUrl)
	if err != nil {
		return fmt.Errorf("failed to parse url[%s]: %s", publicUrl, err)
	}
	publicServerUrl = urlInstance.String()
	return nil
}

var serviceLogs *chan<- common.ServiceLog

func SetServiceLogs(instance *chan<- common.ServiceLog) {
	serviceLogs = instance
}
