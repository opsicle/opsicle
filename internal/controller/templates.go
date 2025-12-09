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
	"opsicle/internal/types"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

func registerAutomationTemplatesRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1/templates").Subrouter()
	v1.Handle("", requiresAuth(http.HandlerFunc(handleListTemplatesV1))).Methods(http.MethodGet)

	templateRouter := opts.Router.PathPrefix("/v1/template").Subrouter()
	templateRouter.Handle("/{templateId}", requiresAuth(http.HandlerFunc(handleDeleteTemplateV1))).Methods(http.MethodDelete)
	templateRouter.Handle("/{templateId}", requiresAuth(http.HandlerFunc(handleGetTemplateV1))).Methods(http.MethodGet)
	templateRouter.Handle("", requiresAuth(http.HandlerFunc(handleSubmitTemplateV1))).Methods(http.MethodPost)
	templateRouter.Handle("/invitation/{invitationId}", requiresAuth(http.HandlerFunc(handleUpdateTemplateInvitationV1))).Methods(http.MethodPatch)
	templateRouter.Handle("/{templateId}/user", requiresAuth(http.HandlerFunc(handleCreateTemplateUserV1))).Methods(http.MethodPost)
	templateRouter.Handle("/{templateId}/users", requiresAuth(http.HandlerFunc(handleListTemplateUsersV1))).Methods(http.MethodGet)
	templateRouter.Handle("/{templateId}/user/{userId}", requiresAuth(http.HandlerFunc(handleDeleteTemplateUsersV1))).Methods(http.MethodDelete)
	templateRouter.Handle("/{templateId}/versions", requiresAuth(http.HandlerFunc(handleListTemplateVersionsV1))).Methods(http.MethodGet)
	templateRouter.Handle("/{templateId}/version", requiresAuth(http.HandlerFunc(handleUpdateTemplateVersionV1))).Methods(http.MethodPut)
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
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", types.ErrorInvalidInput)
		return
	}
	var input handleSubmitTemplateV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", types.ErrorInvalidInput)
		return
	}

	var template automations.Template
	if err := yaml.Unmarshal(input.Data, &template); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse automation tempalte data", types.ErrorInvalidInput)
		return
	}

	automationTemplateVersion, err := models.CreateTemplateVersionV1(models.CreateTemplateVersionV1Opts{
		Db:       dbInstance,
		Template: template,
		UserId:   session.UserId,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to create automation template in db: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create automation tempalte", types.ErrorDatabaseIssue)
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

type handleDeleteTemplateV1Output struct {
	IsSuccessful bool   `json:"isSuccessful"`
	TemplateId   string `json:"templateId"`
	TemplateName string `json:"templateName"`
}

func handleDeleteTemplateV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
	vars := mux.Vars(r)
	automationTemplateId := vars["templateId"]
	log(common.LogLevelDebug, fmt.Sprintf("user[%s] is deleting automationTemplate[%s]", session.UserId, automationTemplateId))

	template := models.Template{Id: &automationTemplateId}
	isUserAllowedToDoThis, err := template.CanUserDeleteV1(models.DatabaseConnection{Db: dbInstance}, session.UserId)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to load user[%s] of template[%s]: %s", session.UserId, template.GetId(), err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusForbidden, "not found", types.ErrorInsufficientPermissions)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "not allowed to delete template", types.ErrorDatabaseIssue)
		return
	}
	if !isUserAllowedToDoThis {
		log(common.LogLevelError, fmt.Sprintf("user[%s] not authorized to delete template[%s]", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "not allowed to delete template", types.ErrorInsufficientPermissions)
		return
	}
	if err := template.LoadV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to load template[%s]: %s", template.GetId(), err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "template not found", types.ErrorNotFound)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "template could not be retrieved", types.ErrorDatabaseIssue)
		return
	}
	if err := template.DeleteV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to delete template[%s]: %s", template.GetId(), err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "template could not be retrieved", types.ErrorDatabaseIssue)
		return
	}

	output := handleDeleteTemplateV1Output{
		IsSuccessful: true,
		TemplateId:   template.GetId(),
		TemplateName: template.GetName(),
	}
	log(common.LogLevelInfo, fmt.Sprintf("user[%s] deleted template[%s]: %s", session.UserId, template.GetId(), err))
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
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", types.ErrorInvalidInput)
		return
	}
	var input handleListTemplatesV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", types.ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("retrieving automation templates for user[%s]", session.UserId))

	user := models.User{Id: &session.UserId}
	templates, err := user.ListTemplatesV1(models.DatabaseConnection{Db: dbInstance})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to list automation templates", types.ErrorDatabaseIssue)
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

type handleUpdateTemplateInvitationV1Output struct {
	IsSuccessful bool                                        `json:"isSuccessful"`
	TemplateUser *handleUpdateTemplateInvitationV1OutputUser `json:"templateUser,omitempty"`
}

type handleUpdateTemplateInvitationV1OutputUser struct {
	UserId       *string `json:"userId"`
	UserEmail    *string `json:"userEmail"`
	TemplateId   *string `json:"templateId"`
	TemplateName *string `json:"templateName"`
	CanView      bool    `json:"canView"`
	CanExecute   bool    `json:"canExecute"`
	CanUpdate    bool    `json:"canUpdate"`
	CanDelete    bool    `json:"canDelete"`
	CanInvite    bool    `json:"canInvite"`
	CreatedBy    *string `json:"createdBy"`
}

type handleUpdateTemplateInvitationV1Input struct {
	IsAcceptance bool   `json:"isAcceptance"`
	JoinCode     string `json:"joinCode"`
}

func handleUpdateTemplateInvitationV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
	vars := mux.Vars(r)
	invitationId := vars["invitationId"]
	log(common.LogLevelInfo, fmt.Sprintf("user[%s] is updating templateInvitation[%s]", session.UserId, invitationId))
	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", types.ErrorInvalidInput)
		return
	}

	var input handleUpdateTemplateInvitationV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", types.ErrorInvalidInput)
		return
	}

	templateUserInvitation := models.TemplateUserInvitation{Id: invitationId}
	if err := templateUserInvitation.LoadV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed find invitation", types.ErrorInvitationInvalid)
			return
		}
		log(common.LogLevelError, fmt.Sprintf("failed to load invitation: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to load", types.ErrorInvalidInput)
		return
	}

	if !strings.EqualFold(*templateUserInvitation.AcceptorId, session.UserId) {
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "user not allowed", types.ErrorInvitationInvalid)
		return
	}
	if strings.Compare(templateUserInvitation.JoinCode, input.JoinCode) != 0 {
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "invalid join code", types.ErrorInvitationInvalid)
		return
	}

	if input.IsAcceptance {
		template := models.Template{Id: &templateUserInvitation.TemplateId}
		if err := template.AddUserV1(models.AddUserToTemplateV1{
			Db:         dbInstance,
			UserId:     session.UserId,
			CanView:    templateUserInvitation.CanView,
			CanExecute: templateUserInvitation.CanExecute,
			CanUpdate:  templateUserInvitation.CanUpdate,
			CanDelete:  templateUserInvitation.CanDelete,
			CanInvite:  templateUserInvitation.CanInvite,
			CreatedBy:  templateUserInvitation.InviterId,
		}); err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to add user[%s] to template[%s]: %s", *templateUserInvitation.AcceptorId, templateUserInvitation.TemplateId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to add user", types.ErrorDatabaseIssue)
			return
		}

		templateUser := models.TemplateUser{
			TemplateId: &templateUserInvitation.TemplateId,
			UserId:     &session.UserId,
		}
		if err := templateUser.LoadV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to retrieve user[%s] of template[%s]: %s", session.UserId, templateUserInvitation.TemplateId, err))
		}

		output := handleUpdateTemplateInvitationV1Output{
			IsSuccessful: true,
			TemplateUser: &handleUpdateTemplateInvitationV1OutputUser{
				UserId:       templateUser.UserId,
				UserEmail:    templateUser.UserEmail,
				TemplateId:   templateUser.TemplateId,
				TemplateName: templateUser.TemplateName,
				CanView:      templateUser.CanView,
				CanExecute:   templateUser.CanExecute,
				CanDelete:    templateUser.CanDelete,
				CanUpdate:    templateUser.CanUpdate,
				CanInvite:    templateUser.CanInvite,
				CreatedBy:    templateUser.CreatedBy.Id,
			},
		}
		audit.Log(audit.LogEntry{
			EntityId:     session.UserId,
			EntityType:   audit.UserEntity,
			Verb:         audit.Accept,
			ResourceId:   templateUserInvitation.Id,
			ResourceType: audit.TemplateUserInvitationResource,
			Status:       audit.Success,
			SrcIp:        &session.SourceIp,
			SrcUa:        &session.UserAgent,
			DstHost:      &r.Host,
		})
		if err := templateUserInvitation.DeleteByIdV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to delete invitation", types.ErrorDatabaseIssue)
			return
		}
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
		return
	} else {
		if err := templateUserInvitation.DeleteByIdV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to delete invitation", types.ErrorDatabaseIssue)
			return
		}
		audit.Log(audit.LogEntry{
			EntityId:     session.UserId,
			EntityType:   audit.UserEntity,
			Verb:         audit.Reject,
			ResourceId:   templateUserInvitation.Id,
			ResourceType: audit.TemplateUserInvitationResource,
			Status:       audit.Success,
			SrcIp:        &session.SourceIp,
			SrcUa:        &session.UserAgent,
			DstHost:      &r.Host,
		})
		output := handleUpdateTemplateInvitationV1Output{
			IsSuccessful: true,
		}
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
		return
	}

	if err := templateUserInvitation.DeleteByIdV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to delete invitation", types.ErrorDatabaseIssue)
		return
	}

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "tmp ok", nil)

	// 	orgUser, err := org.GetUserV1(models.GetOrgUserV1Opts{
	// 		Db:     dbInstance,
	// 		UserId: session.UserId,
	// 	})
	// 	if err != nil {
	// 		if errors.Is(err, models.ErrorNotFound) {
	// 			common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org user", types.ErrorDatabaseIssue)
	// 			return
	// 		}
	// 		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org user", types.ErrorDatabaseIssue)
	// 		return
	// 	}

	// 	output := handleUpdateOrgInvitationV1Output{
	// 		JoinedAt:       orgUser.JoinedAt,
	// 		MembershipType: orgUser.MemberType,
	// 		OrgId:          orgUser.OrgId,
	// 		OrgCode:        orgUser.OrgCode,
	// 		OrgName:        orgUser.OrgName,
	// 		UserId:         orgUser.UserId,
	// 	}
	// 	audit.Log(audit.LogEntry{
	// 		EntityId:     session.UserId,
	// 		EntityType:   audit.UserEntity,
	// 		Verb:         audit.Update,
	// 		ResourceId:   orgInvite.Id,
	// 		ResourceType: audit.OrgUserInvitationResource,
	// 		Status:       audit.Success,
	// 		SrcIp:        &session.SourceIp,
	// 		SrcUa:        &session.UserAgent,
	// 		DstHost:      &r.Host,
	// 	})
	// 	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
	// } else {
	// 	if err := orgInvite.DeleteById(models.DatabaseConnection{Db: dbInstance}); err != nil {
	// 		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to delete invitation", types.ErrorDatabaseIssue)
	// 		return
	// 	}
	// 	audit.Log(audit.LogEntry{
	// 		EntityId:     session.UserId,
	// 		EntityType:   audit.UserEntity,
	// 		Verb:         audit.Delete,
	// 		ResourceId:   orgInvite.Id,
	// 		ResourceType: audit.OrgUserInvitationResource,
	// 		Status:       audit.Success,
	// 		SrcIp:        &session.SourceIp,
	// 		SrcUa:        &session.UserAgent,
	// 		DstHost:      &r.Host,
	// 	})
	// 	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", nil)
	// }
}

type handleCreateTemplateUserV1Output struct {
	Id             string `json:"id"`
	JoinCode       string `json:"joinCode"`
	IsExistingUser bool   `json:"isExistingUser"`
}

type handleCreateTemplateUserV1Input struct {
	UserId     *string `json:"userId"`
	UserEmail  *string `json:"userEmail"`
	CanView    bool    `json:"canView"`
	CanExecute bool    `json:"canExecute"`
	CanUpdate  bool    `json:"canUpdate"`
	CanDelete  bool    `json:"canDelete"`
	CanInvite  bool    `json:"canInvite"`
}

func handleCreateTemplateUserV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
	vars := mux.Vars(r)
	templateId := vars["templateId"]
	log(common.LogLevelInfo, fmt.Sprintf("user[%s] is listing template[%s] users", session.UserId, templateId))

	if _, err := uuid.Parse(templateId); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid template id", types.ErrorInvalidInput)
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to read body: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to get body data", types.ErrorInvalidInput)
		return
	}
	var input handleCreateTemplateUserV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to marshal body: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", types.ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("inviting user to join template[%s]", templateId))

	isUserExists := false
	isInputUuid := input.UserId != nil
	isInputEmail := input.UserEmail != nil
	var userInstance models.User
	var userLoadError error
	if isInputUuid {
		userInstance.Id = input.UserId
		userLoadError = userInstance.LoadByIdV1(models.DatabaseConnection{Db: dbInstance})
	} else if isInputEmail {
		userInstance.Email = *input.UserEmail
		userLoadError = userInstance.LoadByEmailV1(models.DatabaseConnection{Db: dbInstance})
	} else {
		log(common.LogLevelError, "failed to receive either an id or email")
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to receive either an id or email", types.ErrorInvalidInput)
		return
	}
	if userLoadError == nil {
		log(common.LogLevelInfo, fmt.Sprintf("inviting existing user[%s] to template[%s]", userInstance.GetId(), templateId))
		isUserExists = true
	} else if isInputUuid || !isInputEmail {
		log(common.LogLevelError, fmt.Sprintf("failed to retrieve user: %s", userLoadError))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to retrieve user", types.ErrorInvalidInput)
		return
	} else if isInputEmail {
		log(common.LogLevelInfo, fmt.Sprintf("inviting future user[%s] to template[%s]", *input.UserEmail, templateId))
	}

	templateInstance := models.Template{Id: &templateId}
	log(common.LogLevelDebug, fmt.Sprintf("verifying if user[%s] can invite other users", session.UserId))
	canUserInvite, err := templateInstance.CanUserInviteV1(models.DatabaseConnection{Db: dbInstance}, session.UserId)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to get user[%s] permissions on template[%s]: %s", session.UserId, templateId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to check user permissions", types.ErrorNotFound)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to check user permissions", types.ErrorDatabaseIssue)
		return
	} else if !canUserInvite {
		log(common.LogLevelError, fmt.Sprintf("user[%s] cannot invite other users to template[%s]", session.UserId, templateId))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to check user permissions", types.ErrorInsufficientPermissions)
		return
	}

	joinCode, err := common.GenerateRandomString(32)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to generate random string: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to generate join code", err)
		return
	}
	inviteOpts := models.InviteTemplateUserV1Opts{
		Db:         dbInstance,
		InviterId:  session.UserId,
		JoinCode:   joinCode,
		CanView:    input.CanView,
		CanUpdate:  input.CanUpdate,
		CanExecute: input.CanExecute,
		CanDelete:  input.CanDelete,
		CanInvite:  input.CanInvite,
	}
	if isUserExists {
		inviteOpts.AcceptorId = userInstance.Id
	} else if isInputEmail {
		inviteOpts.AcceptorEmail = input.UserEmail
	}

	log(common.LogLevelDebug, "creating template invitation...")
	invitation, err := templateInstance.InviteUserV1(inviteOpts)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to invite user to template[%s]: %s", templateId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to invite user to template", err)
		return
	}

	output := handleCreateTemplateUserV1Output{
		Id:             invitation.InvitationId,
		IsExistingUser: invitation.IsExistingUser,
		JoinCode:       joinCode,
	}
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}

type handleListTemplateUsersV1Output struct {
	Users []handleListTemplateUsersV1OutputUser `json:"users"`
}

type handleListTemplateUsersV1OutputUser struct {
	Id            string    `json:"id"`
	Email         string    `json:"email"`
	CanView       bool      `json:"canView"`
	CanExecute    bool      `json:"canExecute"`
	CanUpdate     bool      `json:"canUpdate"`
	CanDelete     bool      `json:"canDelete"`
	CanInvite     bool      `json:"canInvite"`
	CreatedBy     string    `json:"createdBy"`
	CreatedAt     time.Time `json:"createdAt"`
	LastUpdatedBy string    `json:"lastUpdatedBy`
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
}

func handleListTemplateUsersV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
	vars := mux.Vars(r)
	templateId := vars["templateId"]
	log(common.LogLevelInfo, fmt.Sprintf("user[%s] is listing template[%s] users", session.UserId, templateId))

	template := models.Template{Id: &templateId}
	if err := template.ListUsersV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to list template users: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to list template users", types.ErrorDatabaseIssue)
		return
	}
	output := handleListTemplateUsersV1Output{}
	for _, user := range template.Users {
		createdBy := ""
		if user.CreatedBy.Id != nil {
			createdBy = user.CreatedBy.Email
		}
		lastUpdatedBy := ""
		if user.LastUpdatedBy.Id != nil {
			lastUpdatedBy = user.LastUpdatedBy.Email
		}
		output.Users = append(output.Users, handleListTemplateUsersV1OutputUser{
			Id:            user.GetUserId(),
			Email:         user.GetUserEmail(),
			CanView:       user.CanView,
			CanExecute:    user.CanExecute,
			CanUpdate:     user.CanUpdate,
			CanDelete:     user.CanDelete,
			CanInvite:     user.CanInvite,
			CreatedAt:     user.CreatedAt,
			CreatedBy:     createdBy,
			LastUpdatedAt: user.LastUpdatedAt,
			LastUpdatedBy: lastUpdatedBy,
		})
	}

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}

type handleDeleteTemplateUsersV1Output struct {
	IsSuccessful bool `json:"isSuccessful"`
}

func handleDeleteTemplateUsersV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
	vars := mux.Vars(r)
	templateId := vars["templateId"]
	userId := vars["userId"]
	log(common.LogLevelInfo, fmt.Sprintf("user[%s] is removing user[%s] from template[%s]", session.UserId, userId, templateId))

	template := models.Template{Id: &templateId}
	if err := template.ListUsersV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to list template users: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to list template users", types.ErrorDatabaseIssue)
		return
	}
	if len(template.Users) <= 1 {
		log(common.LogLevelError, "last user cannot be removed")
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "last user cannot be removed", types.ErrorLastUserInResource)
		return
	}

	var userToBeDeleted *models.TemplateUser = nil
	managingUsers := []string{}
	for _, templateUser := range template.Users {
		if templateUser.GetUserId() == userId {
			userToBeDeleted = &templateUser
		}
		if templateUser.CanInvite {
			managingUsers = append(managingUsers, *templateUser.UserId)
		}
	}
	if userToBeDeleted == nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] was not found", userId))
		common.SendHttpFailResponse(w, r, http.StatusNotFound, "user not found", types.ErrorNotFound)
		return
	}
	if len(managingUsers) == 1 && managingUsers[0] == userId {
		log(common.LogLevelError, "last user with invite permissions cannot be removed")
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "last user with invite permissions cannot be removed", types.ErrorLastManagerOfResource)
		return
	}
	canInvite, err := template.CanUserInviteV1(models.DatabaseConnection{Db: dbInstance}, session.UserId)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to check if user[%s] can invite other users: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "user cannot be removed", types.ErrorDatabaseIssue)
		return
	} else if !canInvite {
		log(common.LogLevelError, fmt.Sprintf("user[%s] not allowed to manage users of this template", session.UserId))
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "user cannot be removed", types.ErrorInsufficientPermissions)
		return
	}

	if err := userToBeDeleted.DeleteV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to remove user[%s]: %s", userId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to remove user", types.ErrorDatabaseIssue)
		return
	}

	output := handleDeleteTemplateUsersV1Output{IsSuccessful: true}
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
	canUpdate, err := template.CanUserUpdateV1(models.DatabaseConnection{Db: dbInstance}, session.UserId)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to check if user can perform update: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to list template versions", types.ErrorDatabaseIssue)
		return
	} else if !canUpdate {
		log(common.LogLevelError, fmt.Sprintf("requesting user with id '%s' is not able to perform updates", session.UserId))
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to list template versions", types.ErrorInsufficientPermissions)
		return
	}
	if err := template.LoadV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to load template: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to list template versions", types.ErrorDatabaseIssue)
		return
	}
	if err := template.LoadVersionsV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to load versions: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to list template versions", types.ErrorDatabaseIssue)
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
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid template id", types.ErrorInvalidInput)
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to read body: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to get body data", types.ErrorInvalidInput)
		return
	}
	var input handleUpdateTemplateVersionV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to marshal body: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", types.ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("updating default version of template[%s] to version[%v]", templateId, input.Version))

	template := models.Template{Id: &templateId}
	userCanUpdate, err := template.CanUserUpdateV1(models.DatabaseConnection{Db: dbInstance}, session.UserId)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to get permissions for user[%s] : %s", session.UserId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to get user permissions", types.ErrorNotFound)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to get user permissions", types.ErrorDatabaseIssue)
		return
	} else if !userCanUpdate {
		log(common.LogLevelError, fmt.Sprintf("user[%s] not authorized to update template[%s]", session.UserId, templateId))
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to get user permissions", types.ErrorInsufficientPermissions)
		return
	}
	if err := template.UpdateFieldsV1(models.UpdateFieldsV1{
		Db: dbInstance,
		FieldsToSet: map[string]any{
			"version": input.Version,
		},
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to process update to default version of template[%s]: %s", templateId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to update default version", types.ErrorDatabaseIssue)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("updated default version of template[%s] to %v", templateId, input.Version))
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleUpdateTemplateVersionV1Output{Version: input.Version})
}
