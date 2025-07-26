package common

import "time"

const (
	DefaultDurationConnectionTimeout = 10 * time.Second
)

type LogLevel string

const (
	LogLevelTrace LogLevel = "trace"
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

var LogLevels = []LogLevel{
	LogLevelTrace,
	LogLevelDebug,
	LogLevelInfo,
	LogLevelWarn,
	LogLevelError,
}

const (
	RuntimeDocker     = "docker"
	RuntimeKubernetes = "kubernetes"
)

var Runtimes = []string{
	RuntimeDocker,
	RuntimeKubernetes,
}

const (
	StorageDatabase   = "database"
	StorageFilesystem = "filesystem"
)

var Storages = []string{
	StorageDatabase,
	StorageFilesystem,
}
