package queue

import (
	"opsicle/internal/common"
)

var isNoopInUse bool
var noopServiceLog chan common.ServiceLog

func initNoopServiceLog() {
	noopServiceLog = make(chan common.ServiceLog, 32)
}

func startNoopServiceLog() {
	isNoopInUse = true
	for {
		_, ok := <-noopServiceLog
		if !ok {
			break
		}
	}
}

func stopNoopServiceLog() {
	if isNoopInUse {
		close(noopServiceLog)
	}
}
