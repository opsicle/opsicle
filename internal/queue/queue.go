package queue

var fifoQueue FifoQueue

type FifoQueue interface {
	Push(key string, data any)
	Pop(key string) any
}

func GetFifoQueue() FifoQueue {
	return fifoQueue
}
