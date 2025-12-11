package coordinator

import (
	"net/http"
)

func registerInitRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)
	v1 := opts.Router.PathPrefix("/v1").Subrouter()
	v1.Handle("/init", requiresAuth(http.HandlerFunc(handleWorkerRegistrationV1))).Methods(http.MethodGet)
	v1.Handle("/status", requiresAuth(http.HandlerFunc(handleWorkerStatusRetrievalV1))).Methods(http.MethodGet)
}

func handleWorkerRegistrationV1(w http.ResponseWriter, r *http.Request) {

}

func handleWorkerStatusRetrievalV1(w http.ResponseWriter, r *http.Request) {

}
