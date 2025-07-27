package controller

import (
	"opsicle/internal/common"

	"github.com/gorilla/mux"
)

type commonHttpResponse common.HttpResponse

type RouteRegistrationOpts struct {
	Router      *mux.Router
	ServiceLogs chan<- common.ServiceLog
}
