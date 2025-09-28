package queue

import "errors"

var (
	ErrorClientUndefined          = errors.New("client undefined")
	ErrorStreamingClientUndefined = errors.New("streaming client undefined")
)
