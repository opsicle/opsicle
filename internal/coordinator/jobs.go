package coordinator

import (
	"net/http"
	"opsicle/internal/common"
)

func registerJobsRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)
	v1 := opts.Router.PathPrefix("/v1/jobs").Subrouter()
	v1.Handle("", requiresAuth(http.HandlerFunc(handleGetJobV1))).Methods(http.MethodGet)

}

func handleGetJobV1(w http.ResponseWriter, r *http.Request) {
	// log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	// session := r.Context().Value(authRequestContext).(identity)

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok")
}
