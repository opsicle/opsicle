package queue

var fifo FifoQueue

type FifoQueue interface {
	Push(key string, data any)
	Pop(key string) any
}

func GetFifo() FifoQueue {
	return fifo
}
