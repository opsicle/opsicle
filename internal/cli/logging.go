package cli

import (
	"opsicle/internal/common"

	"github.com/sirupsen/logrus"
)

func InitLogging(logLevel string) {
	switch common.LogLevel(logLevel) {
	case common.LogLevelTrace:
		logrus.SetLevel(logrus.TraceLevel)
	case common.LogLevelDebug:
		logrus.SetLevel(logrus.DebugLevel)
	case common.LogLevelInfo:
		logrus.SetLevel(logrus.InfoLevel)
	case common.LogLevelWarn:
		logrus.SetLevel(logrus.WarnLevel)
	case common.LogLevelError:
		logrus.SetLevel(logrus.ErrorLevel)
	}
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}
