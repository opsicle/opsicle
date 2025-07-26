package controller

import (
	"net/http"
	"opsicle/internal/common"

	"github.com/gorilla/mux"
)

func registerAutomationTemplatesRoutesV1(router *mux.Router, serviceLogs chan<- common.ServiceLog) {
	requiresAuth := getRouteAuther(serviceLogs)

	createAutomationTemplateHandler := getCreateAutomationTemplateV1(serviceLogs)
	router.Handle("", requiresAuth(createAutomationTemplateHandler)).Methods(http.MethodPost)

	listAutomationTemplateHandler := getListAutomationTemplatesV1(serviceLogs)
	router.Handle("", requiresAuth(listAutomationTemplateHandler)).Methods(http.MethodGet)

	getAutomationTemplateHandler := getGetAutomationTemplateV1(serviceLogs)
	router.Handle("/{id}", requiresAuth(getAutomationTemplateHandler)).Methods(http.MethodGet)
}

func getCreateAutomationTemplateV1(serviceLogs chan<- common.ServiceLog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "creating automation template...")
		w.Write([]byte("create"))
	}
}

func getGetAutomationTemplateV1(serviceLogs chan<- common.ServiceLog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		automationTemplateId := vars["id"]
		authDetails, ok := r.Context().Value(authRequestContext).(auth)
		if !ok {
			common.SendHttpFailResponse(w, r, http.StatusTooEarly, "not implemented yet", nil)
			return
		}
		serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "role[%s] requested retrieval of automationTemplate[%s] from organisation[%s]", authDetails.OrganizationRoleId, automationTemplateId, authDetails.OrganizationId)
		common.SendHttpSuccessResponse(w, r, http.StatusTooEarly, "not implemented yet")
	}
}

func getListAutomationTemplatesV1(serviceLogs chan<- common.ServiceLog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("list"))
	}
}
