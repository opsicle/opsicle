package controller

import (
	"net/http"
	"opsicle/internal/common"
)

func registerUtilityRoutes(opts RouteRegistrationOpts) {
	requireApiKey := getInternalRouteAuther(opts.ApiKeys, opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1").Subrouter()

	v1.Handle("/healthz", requireApiKey(http.HandlerFunc(handleVerifyApiKeyV1))).Methods(http.MethodGet)
}

func handleVerifyApiKeyV1(w http.ResponseWriter, r *http.Request) {
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok")
}
