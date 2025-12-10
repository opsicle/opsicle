package controller

import (
	"opsicle/internal/common"

	"github.com/gorilla/mux"
)

type commonHttpResponse common.HttpResponse

type RouteRegistrationOpts struct {
	// ApiKey is used only for certain handler groups
	ApiKeys []string

	// Router is the internal implementation of http.Handler
	Router *mux.Router

	// ServiceLogs is a channel where logs should be sent to for
	// processing by a central logs mechanism
	ServiceLogs chan<- common.ServiceLog
}
