package controller

import (
	"net/http"
	"opsicle/internal/common"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func GetHttpApplication(
	serviceLogs chan<- common.ServiceLog,
) http.Handler {
	handler := mux.NewRouter()
	handler.NotFoundHandler = common.GetNotFoundHandler()

	handler.Handle("/metrics", promhttp.Handler())
	api := handler.PathPrefix("/api").Subrouter()
	apiV1 := api.PathPrefix("/v1").Subrouter()

	registerAutomationTemplatesRoutesV1(apiV1.PathPrefix("/automation-templates").Subrouter(), serviceLogs)
	registerSessionRoutesV1(apiV1.PathPrefix("/session").Subrouter(), serviceLogs)

	return handler
}
