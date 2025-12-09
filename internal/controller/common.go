package controller

import (
	"database/sql"
	"net/url"
	"opsicle/internal/cache"
	"opsicle/internal/common"
	"opsicle/internal/queue"
)

const (
	sessionCachePrefix = "session"
)

var apiKeys []string
var cacheInstance cache.Cache
var dbInstance *sql.DB
var publicServerUrl *url.URL
var queueInstance queue.Instance
var serviceLogs *chan<- common.ServiceLog
