package common

import "time"

const (
	DefaultDurationConnectionTimeout = 10 * time.Second
)

const (
	LogLevelTrace = "trace"
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

var LogLevels = []string{
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
