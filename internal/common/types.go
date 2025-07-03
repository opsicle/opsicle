package common

import (
	"opsicle/internal/config"

	"github.com/sirupsen/logrus"
)

type Done struct{}

type Resource struct {
	ApiVersion string   `json:"apiVersion" yaml:"apiVersion"`
	Type       string   `json:"type" yaml:"type"`
	Metadata   Metadata `json:"metadata" yaml:"metadata"`
}

type Metadata struct {
	Name        string            `json:"name" yaml:"name"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type AutomationLog struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

type ServiceLog struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

func StartServiceLogLoop(serviceLogs chan ServiceLog) {
	go func() {
		for {
			logEntry, ok := <-serviceLogs
			if !ok {
				return
			}
			log := logrus.Info
			switch logEntry.Level {
			case config.LogLevelTrace:
				log = logrus.Trace
			case config.LogLevelDebug:
				log = logrus.Debug
			case config.LogLevelInfo:
				log = logrus.Info
			case config.LogLevelWarn:
				log = logrus.Warn
			case config.LogLevelError:
				log = logrus.Error
			}
			log(logEntry.Message)
		}
	}()
}
