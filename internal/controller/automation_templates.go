package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/audit"
	"opsicle/internal/automations"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

func registerAutomationTemplatesRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1/automation-templates").Subrouter()

	v1.Handle("", requiresAuth(http.HandlerFunc(handleSubmitAutomationTemplateV1))).Methods(http.MethodPost)
	v1.Handle("", requiresAuth(http.HandlerFunc(listAutomationTemplatesHandlerV1))).Methods(http.MethodGet)
	v1.Handle("/{id}", requiresAuth(http.HandlerFunc(getAutomationTemplateHandlerV1))).Methods(http.MethodGet)
}

type handleSubmitAutomationTemplateV1Output struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Version int64  `json:"version"`
}

type handleSubmitAutomationTemplateV1Input struct {
	Data []byte `json:"data"`
}

func handleSubmitAutomationTemplateV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
	log(common.LogLevelDebug, fmt.Sprintf("user[%s] is creating an automation template", session.UserId))

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", ErrorInvalidInput)
		return
	}
	var input handleSubmitAutomationTemplateV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", ErrorInvalidInput)
		return
	}

	var template automations.Template
	if err := yaml.Unmarshal(input.Data, &template); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse automation tempalte data", ErrorInvalidInput)
		return
	}

	automationTemplateVersion, err := models.SubmitAutomationTemplateV1(models.SubmitAutomationTemplateV1Opts{
		Db:       db,
		Template: template,
		UserId:   session.UserId,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to create automation template in db: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create automation tempalte", ErrorDatabaseIssue)
		return
	}
	output := handleSubmitAutomationTemplateV1Output{
		Id:      *automationTemplateVersion.Id,
		Name:    *automationTemplateVersion.Name,
		Version: *automationTemplateVersion.Version,
	}

	verb := audit.Create
	if *automationTemplateVersion.Version > 1 {
		verb = audit.Update
	}
	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         verb,
		ResourceId:   *automationTemplateVersion.Id,
		ResourceType: audit.AutomationTemplateResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
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
	log(common.LogLevelDebug, fmt.Sprintf("user[%s] requested retrieval of automationTemplate[%s]", currentUser.UserId, automationTemplateId))
	common.SendHttpSuccessResponse(w, r, http.StatusTooEarly, "not implemented yet")
}

func listAutomationTemplatesHandlerV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelInfo, "this endpoint lists automation templates")
	w.Write([]byte("list"))
}
