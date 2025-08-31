package cache

import (
	"opsicle/internal/common"
)

var noopServiceLog chan common.ServiceLog

func initNoopServiceLog() {
	noopServiceLog = make(chan common.ServiceLog, 32)
}

func startNoopServiceLog() {
	for {
		_, ok := <-noopServiceLog
		if !ok {
			break
		}
	}
}

func stopNoopServiceLog() {
	close(noopServiceLog)
}
