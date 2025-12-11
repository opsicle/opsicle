package coordinator

import (
	"net/url"
	"opsicle/internal/cache"
	"opsicle/internal/common"
	"opsicle/internal/queue"
)

var apiKeys []string
var controllerUrl *url.URL
var cacheInstance cache.Cache
var queueInstance queue.Instance
var serviceLogs *chan<- common.ServiceLog
