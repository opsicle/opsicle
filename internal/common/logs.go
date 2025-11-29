package common

var noopServiceLog chan ServiceLog

func init() {
	noopServiceLog = make(chan ServiceLog, 64)
	go startNoopServiceLog()
}

func GetNoopServiceLog() chan ServiceLog {
	return noopServiceLog
}

func startNoopServiceLog() {
	for {
		_, ok := <-noopServiceLog
		if !ok {
			break
		}
	}
}

func StopNoopServiceLog() {
	close(noopServiceLog)
}
