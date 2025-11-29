package persistence

import (
	"sync"
	"time"
)

type statusCode string

const (
	StatusCodeConnectError statusCode = "connect_error"
	StatusCodeInitialising statusCode = "init"
	StatusCodeShuttingDown statusCode = "shutdown"
	StatusCodeOk           statusCode = "ok"
	StatusCodePingError    statusCode = "ping_error"
)

type Status struct {
	code          statusCode
	lastChangedAt time.Time
	lastUpdatedAt time.Time
	err           error
	mutex         sync.Mutex
}

func (ms *Status) GetCode() statusCode {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	return ms.code
}

func (ms *Status) GetError() error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	return ms.err
}

func (ms *Status) GetLastChangedAt() time.Time {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	return ms.lastChangedAt
}

func (ms *Status) GetLastUpdatedAt() time.Time {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	return ms.lastUpdatedAt
}

func (ms *Status) set(code statusCode, err error) {
	ms.mutex.Lock()
	if code != ms.code {
		ms.lastChangedAt = time.Now()
	}
	ms.code = code
	ms.err = err
	ms.lastUpdatedAt = time.Now()
	ms.mutex.Unlock()
}
