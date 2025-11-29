package coordinator

import (
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/persistence"
	"time"

	"github.com/gorilla/mux"
)

type HttpApplicationOpts struct {
	Cache *persistence.Redis
	Queue *persistence.Nats

	LivenessChecks  []func() error
	ReadinessChecks []func() error

	ServiceLogs chan<- common.ServiceLog
}

func GetHttpApplication(opts HttpApplicationOpts) http.Handler {
	handler := mux.NewRouter()

	handler.Handle("/asd", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-time.After(3 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	handler.NotFoundHandler = common.GetNotFoundHandler()
	common.RegisterCommonHttpEndpoints(common.CommonHttpEndpointsOpts{
		LivenessChecks:  opts.LivenessChecks,
		ReadinessChecks: opts.ReadinessChecks,
		Router:          handler,
		ServiceLogs:     opts.ServiceLogs,
	})
	return handler
}
