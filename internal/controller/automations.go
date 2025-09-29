package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/automations"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"
	"opsicle/internal/validate"

	"github.com/gorilla/mux"
)

func registerAutomationRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1/automation").Subrouter()

	v1.Handle("", requiresAuth(http.HandlerFunc(handleCreateAutomationV1))).Methods(http.MethodPost)
	v1.Handle("/{automationId}", requiresAuth(http.HandlerFunc(handleRunAutomationV1))).Methods(http.MethodPost)
}

type CreateAutomationV1OutputData struct {
	AutomationId     string                              `json:"automationId"`
	TemplateId       string                              `json:"templateId"`
	TemplateName     string                              `json:"templateName"`
	TriggeredById    string                              `json:"triggeredById"`
	TriggeredByEmail string                              `json:"triggeredByEmail"`
	VariableMap      map[string]automations.VariableSpec `json:"variableMap"`
}

type CreateAutomationV1Input struct {
	Comment    string  `json:"comment"`
	OrgId      *string `json:"orgId"`
	TemplateId string  `json:"templateId"`
}

func handleCreateAutomationV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", ErrorInvalidInput)
		return
	}
	var input CreateAutomationV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("user[%s] is preparing to execute automation using template[%s]", session.UserId, input.TemplateId))

	templateId := input.TemplateId
	if err := validate.Uuid(templateId); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid template id", ErrorInvalidInput)
		return
	}

	template, err := models.GetTemplateV1(models.GetTemplateV1Opts{
		Db:         db,
		TemplateId: &templateId,
		UserId:     session.UserId,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to get template[%s]: %s", templateId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid template", ErrorInvalidInput)
		return
	}
	if canExecute, err := template.CanUserExecuteV1(models.DatabaseConnection{Db: db}, session.UserId); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to check permissions of user[%s] on template[%s]: %s", session.UserId, templateId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "not allowed", ErrorDatabaseIssue)
		return
	} else if !canExecute {
		log(common.LogLevelError, fmt.Sprintf("user[%s] is not allowed to execute template[%s]: %s", session.UserId, templateId, err))
		common.SendHttpFailResponse(w, r, http.StatusUnauthorized, "not allowed", ErrorInsufficientPermissions)
		return
	}

	automationParams, err := template.GetAutomationParamsV1(models.DatabaseConnection{Db: db})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to create pending automation based on template[%s]: %s", templateId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create pending automation", ErrorDatabaseIssue)
		return
	}
	user := models.User{Id: &session.UserId}
	if err := user.LoadByIdV1(models.DatabaseConnection{Db: db}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to retrieve user[%s]: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "db query failed", ErrorDatabaseIssue)
		return
	}
	redactedUser := user.GetRedacted()
	automationParams.TriggeredBy = &redactedUser
	pendingAutomation, err := models.CreatePendingAutomationV1(models.CreatePendingAutomationV1Opts{
		OrgId:            input.OrgId,
		TemplateContent:  template.Content,
		TemplateId:       template.GetId(),
		TemplateVersion:  template.GetVersion(),
		TriggeredBy:      session.UserId,
		TriggererComment: input.Comment,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to create automation based on template[%s]: %s", templateId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create automation", ErrorDatabaseIssue)
		return
	}

	sourceTemplate, err := pendingAutomation.GetTemplate()
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to parse template content for automation[%s]: %s", *pendingAutomation.Id, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to parse template", ErrorInvalidTemplate)
		return
	}
	variableMap := sourceTemplate.GetVariables()

	output := CreateAutomationV1OutputData{
		AutomationId:     *pendingAutomation.Id,
		TemplateId:       pendingAutomation.TemplateId,
		TemplateName:     sourceTemplate.GetName(),
		TriggeredById:    user.GetId(),
		TriggeredByEmail: user.Email,
		VariableMap:      variableMap,
	}

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}

type RunAutomationV1Output struct {
	AutomationId    string `json:"automationId"`
	AutomationRunId string `json:"automationRunId"`
	IsSuccessful    bool   `json:"isSuccessful"`
}

type RunAutomationV1VariableMap map[string]any

type RunAutomationV1Input struct {
	AutomationId string                     `json:"-"`
	OrgId        *string                    `json:"orgId"`
	VariableMap  RunAutomationV1VariableMap `json:"variableMap"`
}

func handleRunAutomationV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)

	vars := mux.Vars(r)
	automationId := vars["automationId"]

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", ErrorInvalidInput)
		return
	}
	var input RunAutomationV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("user[%s] is executing automation[%s]", session.UserId, automationId))
	var orgId *string = nil
	if input.OrgId != nil {
		if err := validate.Uuid(*input.OrgId); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to validate org id", ErrorInvalidInput)
			return
		}
		orgId = input.OrgId
	}

	automation := models.Automation{Id: &automationId}
	if err := automation.LoadPendingV1(models.DatabaseConnection{Db: db}); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve automation", ErrorInvalidInput)
		return
	}
	queueRunOpts := models.QueueAutomationRunV1Opts{
		Db: db,
		Q:  q,

		Input: input.VariableMap,
		OrgId: orgId,
	}
	queuedAutomationRun, err := automation.RunV1(queueRunOpts)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to run automation: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to insert automation into the queue", ErrorQueueIssue)
		return
	}
	output := RunAutomationV1Output{
		AutomationId:    automationId,
		AutomationRunId: queuedAutomationRun.AutomationRunId,
		IsSuccessful:    false,
	}
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}
