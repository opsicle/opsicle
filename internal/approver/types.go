package approver

import (
	"opsicle/internal/common"

	"github.com/gorilla/mux"
)

type RouteRegistrationOpts struct {
	Router      *mux.Router
	ServiceLogs chan<- common.ServiceLog
}
