package persistence

import (
	"os"
	"time"
)

const DefaultHealthcheckInterval = 3 * time.Second
const DefaultRetryInterval = 3 * time.Second

func getAppName(appName string) string {
	if appName != "" {
		return appName
	}
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown_host"
	}
	return hostname
}
