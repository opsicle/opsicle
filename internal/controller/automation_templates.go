package controller

import (
	"fmt"
	"net/http"
	"opsicle/internal/common"

	"github.com/gorilla/mux"
)

func registerAutomationTemplatesRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1/automation-templates").Subrouter()

	v1.Handle("", requiresAuth(http.HandlerFunc(createAutomationTemplateHandlerV1))).Methods(http.MethodPost)
	v1.Handle("", requiresAuth(http.HandlerFunc(listAutomationTemplatesHandlerV1))).Methods(http.MethodGet)
	v1.Handle("/{id}", requiresAuth(http.HandlerFunc(getAutomationTemplateHandlerV1))).Methods(http.MethodGet)
}

func createAutomationTemplateHandlerV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelDebug, "this endpoint handles creation of an automation template")
	w.Write([]byte("create"))
}

func getAutomationTemplateHandlerV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	vars := mux.Vars(r)
	automationTemplateId := vars["id"]
	currentUser, ok := r.Context().Value(authRequestContext).(identity)
	if !ok {
		common.SendHttpFailResponse(w, r, http.StatusTooEarly, "not implemented yet", nil)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("role[%s] requested retrieval of automationTemplate[%s] from organisation[%s]", currentUser.OrganizationRoleId, automationTemplateId, currentUser.OrganizationId))
	common.SendHttpSuccessResponse(w, r, http.StatusTooEarly, "not implemented yet")
}

func listAutomationTemplatesHandlerV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelInfo, "this endpoint lists automation templates")
	w.Write([]byte("list"))
}
