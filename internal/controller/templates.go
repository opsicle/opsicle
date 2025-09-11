package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/audit"
	"opsicle/internal/automations"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

func registerAutomationTemplatesRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1/templates").Subrouter()

	v1.Handle("", requiresAuth(http.HandlerFunc(handleListTemplatesV1))).Methods(http.MethodGet)

	v1 = opts.Router.PathPrefix("/v1/template").Subrouter()

	v1.Handle("/{templateId}", requiresAuth(http.HandlerFunc(handleGetTemplateV1))).Methods(http.MethodGet)
	v1.Handle("", requiresAuth(http.HandlerFunc(handleSubmitTemplateV1))).Methods(http.MethodPost)
	v1.Handle("/{templateId}/versions", requiresAuth(http.HandlerFunc(handleListTemplateVersionsV1))).Methods(http.MethodGet)
	v1.Handle("/{templateId}/version", requiresAuth(http.HandlerFunc(handleUpdateTemplateVersionV1))).Methods(http.MethodPut)
}

type handleSubmitTemplateV1Output struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Version int64  `json:"version"`
}

type handleSubmitTemplateV1Input struct {
	Data []byte `json:"data"`
}

func handleSubmitTemplateV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
	log(common.LogLevelDebug, fmt.Sprintf("user[%s] is creating an automation template", session.UserId))

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", ErrorInvalidInput)
		return
	}
	var input handleSubmitTemplateV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", ErrorInvalidInput)
		return
	}

	var template automations.Template
	if err := yaml.Unmarshal(input.Data, &template); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse automation tempalte data", ErrorInvalidInput)
		return
	}

	automationTemplateVersion, err := models.SubmitTemplateV1(models.SubmitTemplateV1Opts{
		Db:       db,
		Template: template,
		UserId:   session.UserId,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to create automation template in db: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create automation tempalte", ErrorDatabaseIssue)
		return
	}
	output := handleSubmitTemplateV1Output{
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

func handleGetTemplateV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	vars := mux.Vars(r)
	automationTemplateId := vars["templateId"]
	currentUser, ok := r.Context().Value(authRequestContext).(identity)
	if !ok {
		common.SendHttpFailResponse(w, r, http.StatusTooEarly, "not implemented yet", nil)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("user[%s] requested retrieval of automationTemplate[%s]", currentUser.UserId, automationTemplateId))
	common.SendHttpSuccessResponse(w, r, http.StatusTooEarly, "not implemented yet")
}

type handleListTemplatesV1Output []handleListTemplatesV1OutputTemplate

type handleListTemplatesV1OutputTemplate struct {
	Id            string                                   `json:"id"`
	Content       []byte                                   `json:"content"`
	Description   string                                   `json:"description"`
	Name          string                                   `json:"name"`
	Version       int64                                    `json:"version"`
	CreatedAt     time.Time                                `json:"createdAt"`
	CreatedBy     *handleListTemplatesV1OutputTemplateUser `json:"createdBy"`
	LastUpdatedAt *time.Time                               `json:"lastUpdatedAt"`
	LastUpdatedBy *handleListTemplatesV1OutputTemplateUser `json:"lastUpdatedBy"`
}

type handleListTemplatesV1OutputTemplateUser struct {
	Id    string `json:"id"`
	Email string `json:"email"`
}

type handleListTemplatesV1Input struct {
	Limit int `json:"limit"`
}

func handleListTemplatesV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelInfo, "this endpoint lists automation templates")
	session := r.Context().Value(authRequestContext).(identity)

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", ErrorInvalidInput)
		return
	}
	var input handleListTemplatesV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("retrieving automation templates for user[%s]", session.UserId))

	user := models.User{Id: &session.UserId}
	templates, err := user.ListTemplatesV1(models.DatabaseConnection{Db: db})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to list automation templates", ErrorDatabaseIssue)
		return
	}
	sort.Slice(templates, func(i, j int) bool {
		iTime := time.Time{}
		if templates[i].LastUpdatedAt != nil {
			iTime = *templates[i].LastUpdatedAt
		}
		jTime := time.Time{}
		if templates[j].LastUpdatedAt != nil {
			jTime = *templates[j].LastUpdatedAt
		}
		return iTime.After(jTime)
	})
	output := handleListTemplatesV1Output{}
	for _, template := range templates {
		outputItem := handleListTemplatesV1OutputTemplate{
			Id:            *template.Id,
			Description:   *template.Description,
			Name:          *template.Name,
			Content:       template.Content,
			Version:       *template.Version,
			CreatedAt:     template.CreatedAt,
			LastUpdatedAt: template.LastUpdatedAt,
		}
		if template.CreatedBy != nil {
			outputItem.CreatedBy = &handleListTemplatesV1OutputTemplateUser{
				Id:    template.CreatedBy.GetId(),
				Email: template.CreatedBy.Email,
			}
		}
		if template.LastUpdatedBy != nil {
			outputItem.LastUpdatedBy = &handleListTemplatesV1OutputTemplateUser{
				Id:    template.LastUpdatedBy.GetId(),
				Email: template.LastUpdatedBy.Email,
			}
		}
		output = append(output, outputItem)
	}

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}

type handleListTemplateVersionsV1Output struct {
	Template handleListTemplateVersionsV1OutputTemplate  `json:"template"`
	Versions []handleListTemplateVersionsV1OutputVersion `json:"versions"`
}

type handleListTemplateVersionsV1OutputTemplate struct {
	Id            string                                  `json:"id"`
	Name          string                                  `json:"name"`
	Description   *string                                 `json:"description"`
	Version       int64                                   `json:"version"`
	CreatedAt     time.Time                               `json:"createdAt"`
	CreatedBy     handleListTemplateVersionsV1OutputUser  `json:"createdBy"`
	LastUpdatedAt *time.Time                              `json:"lastUpdatedAt"`
	LastUpdatedBy *handleListTemplateVersionsV1OutputUser `json:"lastUpdatedBy"`
}

type handleListTemplateVersionsV1OutputVersion struct {
	Content   string                                 `json:"content"`
	CreatedAt time.Time                              `json:"createdAt"`
	CreatedBy handleListTemplateVersionsV1OutputUser `json:"createdBy"`
	Version   int64                                  `json:"version"`
}

type handleListTemplateVersionsV1OutputUser struct {
	Id    string `json:"id"`
	Email string `json:"email"`
}

func handleListTemplateVersionsV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
	vars := mux.Vars(r)
	templateId := vars["templateId"]
	log(common.LogLevelInfo, fmt.Sprintf("listing versions of template[%s]", templateId))

	template := &models.Template{Id: &templateId}
	canUpdate, err := template.CanUserUpdateV1(models.DatabaseConnection{Db: db}, session.UserId)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to check if user can perform update: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to list template versions", ErrorDatabaseIssue)
		return
	} else if !canUpdate {
		log(common.LogLevelError, fmt.Sprintf("requesting user with id '%s' is not able to perform updates", session.UserId))
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to list template versions", ErrorInsufficientPermissions)
		return
	}
	if err := template.LoadV1(models.DatabaseConnection{Db: db}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to load template: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to list template versions", ErrorDatabaseIssue)
		return
	}
	if err := template.LoadVersionsV1(models.DatabaseConnection{Db: db}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to load versions: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to list template versions", ErrorDatabaseIssue)
		return
	}
	output := handleListTemplateVersionsV1Output{
		Versions: []handleListTemplateVersionsV1OutputVersion{},
	}
	output.Template = handleListTemplateVersionsV1OutputTemplate{
		Id:          *template.Id,
		Name:        *template.Name,
		Version:     *template.Version,
		Description: template.Description,
		CreatedAt:   template.CreatedAt,
		CreatedBy: handleListTemplateVersionsV1OutputUser{
			Id:    *template.CreatedBy.Id,
			Email: template.CreatedBy.Email,
		},
	}
	if template.LastUpdatedAt != nil {
		output.Template.LastUpdatedAt = template.LastUpdatedAt
		output.Template.LastUpdatedBy = &handleListTemplateVersionsV1OutputUser{
			Id:    *template.LastUpdatedBy.Id,
			Email: template.LastUpdatedBy.Email,
		}
	}
	for _, version := range template.Versions {
		output.Versions = append(
			output.Versions,
			handleListTemplateVersionsV1OutputVersion{
				Content:   version.Content,
				Version:   version.Version,
				CreatedAt: version.CreatedAt,
				CreatedBy: handleListTemplateVersionsV1OutputUser{
					Id:    *version.CreatedBy.Id,
					Email: version.CreatedBy.Email,
				},
			},
		)
	}

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}

type handleUpdateTemplateVersionV1Output struct {
	Version int64 `json:"version"`
}

type handleUpdateTemplateVersionV1Input struct {
	Version int64 `json:"version"`
}

func handleUpdateTemplateVersionV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
	vars := mux.Vars(r)
	templateId := vars["templateId"]

	if _, err := uuid.Parse(templateId); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid template id", ErrorInvalidInput)
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to read body: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to get body data", ErrorInvalidInput)
		return
	}
	var input handleUpdateTemplateVersionV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to marshal body: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("updating default version of template[%s] to version[%v]", templateId, input.Version))

	template := models.Template{Id: &templateId}
	userCanUpdate, err := template.CanUserUpdateV1(models.DatabaseConnection{Db: db}, session.UserId)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to get permissions for user[%s] : %s", session.UserId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to get user permissions", ErrorNotFound)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to get user permissions", ErrorDatabaseIssue)
		return
	} else if !userCanUpdate {
		log(common.LogLevelError, fmt.Sprintf("user[%s] not authorized to update template[%s]", session.UserId, templateId))
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to get user permissions", ErrorInsufficientPermissions)
		return
	}
	if err := template.UpdateFieldsV1(models.UpdateFieldsV1{
		Db: db,
		FieldsToSet: map[string]any{
			"version": input.Version,
		},
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to process update to default version of template[%s]: %s", templateId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to update default version", ErrorDatabaseIssue)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("updated default version of template[%s] to %v", templateId, input.Version))
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleUpdateTemplateVersionV1Output{Version: input.Version})
}
