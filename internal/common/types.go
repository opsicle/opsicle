package common

import (
	"fmt"
	"time"

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

type Cache interface {
	Set(key string, value string, ttl time.Duration) (err error)
	Get(key string) (value string, err error)
	Scan(prefix string) (keys []string, err error)
	Del(key string) (err error)
}

type AutomationLog struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

type ServiceLog struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

func ServiceLogf(level, text string, f ...any) ServiceLog {
	return ServiceLog{
		Level:   level,
		Message: fmt.Sprintf(text, f...),
	}
}

func StartServiceLogLoop(serviceLogs chan ServiceLog) {
	go func() {
		for {
			serviceLog, ok := <-serviceLogs
			if !ok {
				return
			}
			log := logrus.Info
			switch serviceLog.Level {
			case LogLevelTrace:
				log = logrus.Trace
			case LogLevelDebug:
				log = logrus.Debug
			case LogLevelInfo:
				log = logrus.Info
			case LogLevelWarn:
				log = logrus.Warn
			case LogLevelError:
				log = logrus.Error
			}
			log(serviceLog.Message)
		}
	}()
}
