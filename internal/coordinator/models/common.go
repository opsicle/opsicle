package models

import (
	"net/url"
	"opsicle/internal/cache"
	"opsicle/internal/queue"
)

type CacheConnection struct {
	Cache cache.Cache
}

type ControllerUrl struct {
	Url *url.URL
}

type QueueConnection struct {
	Queue queue.Instance
}
