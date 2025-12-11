package controller

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/audit"
	"opsicle/internal/automations"
	"opsicle/internal/common"
	"opsicle/internal/common/images"
	"opsicle/internal/controller/constants"
	"opsicle/internal/controller/models"
	"opsicle/internal/controller/templates"
	"opsicle/internal/email"
	"opsicle/internal/tls"
	"opsicle/internal/types"
	"opsicle/internal/validate"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

func registerOrgRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1/org").Subrouter()

	v1.Handle("/member/types", requiresAuth(http.HandlerFunc(handleListOrgMemberTypesV1))).Methods(http.MethodGet)
	v1.Handle("", requiresAuth(http.HandlerFunc(handleCreateOrgV1))).Methods(http.MethodPost)
	v1.Handle("/{orgId}", requiresAuth(http.HandlerFunc(handleDeleteOrgV1))).Methods(http.MethodDelete)
	v1.Handle("/{orgRef}", requiresAuth(http.HandlerFunc(handleGetOrgV1))).Methods(http.MethodGet)
	v1.Handle("/{orgId}/member", requiresAuth(http.HandlerFunc(handleCreateOrgUserV1))).Methods(http.MethodPost)
	v1.Handle("/{orgId}/member", requiresAuth(http.HandlerFunc(handleGetOrgCurrentUserV1))).Methods(http.MethodGet)
	v1.Handle("/{orgId}/member", requiresAuth(http.HandlerFunc(handleUpdateOrgUserV1))).Methods(http.MethodPatch)
	v1.Handle("/{orgId}/member", requiresAuth(http.HandlerFunc(handleLeaveOrgV1))).Methods(http.MethodDelete)
	v1.Handle("/{orgId}/member/{userId}", requiresAuth(http.HandlerFunc(handleDeleteOrgUserV1))).Methods(http.MethodDelete)
	v1.Handle("/{orgId}/member/{userId}/can/{action}/{resource}", requiresAuth(http.HandlerFunc(handleCanOrgUserActionV1))).Methods(http.MethodGet)
	v1.Handle("/{orgId}/members", requiresAuth(http.HandlerFunc(handleListOrgUsersV1))).Methods(http.MethodGet)
	v1.Handle("/{orgId}/roles", requiresAuth(http.HandlerFunc(handleListOrgRolesV1))).Methods(http.MethodGet)
	v1.Handle("/{orgId}/template", requiresAuth(http.HandlerFunc(handleSubmitOrgTemplateV1))).Methods(http.MethodPost)
	v1.Handle("/{orgId}/templates", requiresAuth(http.HandlerFunc(handleListOrgTemplatesV1))).Methods(http.MethodGet)
	v1.Handle("/{orgId}/tokens", requiresAuth(http.HandlerFunc(handleListOrgTokensV1))).Methods(http.MethodGet)
	v1.Handle("/{orgId}/token/{tokenId}", requiresAuth(http.HandlerFunc(handleGetOrgTokenV1))).Methods(http.MethodGet)
	v1.Handle("/{orgId}/token", requiresAuth(http.HandlerFunc(handleCreateOrgTokenV1))).Methods(http.MethodPost)
	v1.Handle("/invitation/{invitationId}", requiresAuth(http.HandlerFunc(handleUpdateOrgInvitationV1))).Methods(http.MethodPatch)
	v1.HandleFunc("/validate", handleOrgTokenValidationV1).Methods(http.MethodPost)

	v1 = opts.Router.PathPrefix("/v1/orgs").Subrouter()

	v1.Handle("", requiresAuth(http.HandlerFunc(handleListOrgsV1))).Methods(http.MethodGet)
}

type CreateOrgV1Output struct {
	Id   string `json:"id"`
	Code string `json:"code"`
}

type CreateOrgV1Input struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type DeleteOrgV1Output struct {
	IsSuccessful bool `json:"isSuccessful"`
}

// handleDeleteOrgV1 godoc
// @Summary      Deletes an organisation
// @Description  Deletes the organisation identified by the supplied ID if the requester is authorized to do so
// @Tags         controller-service
// @Produce      json
// @Param        orgId path string true "Organisation ID"
// @Success      200 {object} commonHttpResponse "ok"
// @Failure      400 {object} commonHttpResponse "invalid input"
// @Failure      403 {object} commonHttpResponse "forbidden"
// @Failure      404 {object} commonHttpResponse "not found"
// @Failure      500 {object} commonHttpResponse "internal server error"
// @Router       /api/v1/org/{orgId} [delete]
func handleDeleteOrgV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)

	vars := mux.Vars(r)
	orgId := vars["orgId"]
	if err := validate.Uuid(orgId); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid org id[%s]: %s", session.UserId, orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid org id", types.ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("user[%s] requested deletion of org[%s]", session.UserId, orgId))

	orgInstance, err := models.GetOrgV1(models.GetOrgV1Opts{Db: dbInstance, Id: &orgId})
	if err != nil {
		switch {
		case errors.Is(err, models.ErrorNotFound):
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "org not found", types.ErrorNotFound)
		default:
			log(common.LogLevelError, fmt.Sprintf("failed to load org[%s]: %s", orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to load org", types.ErrorDatabaseIssue)
		}
		return
	}

	orgUser, err := orgInstance.GetUserV1(models.GetOrgUserV1Opts{Db: dbInstance, UserId: session.UserId})
	if err != nil {
		switch {
		case errors.Is(err, models.ErrorNotFound):
			log(common.LogLevelWarn, fmt.Sprintf("user[%s] attempted to delete org[%s] without membership", session.UserId, orgId))
			common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to verify requester", types.ErrorInsufficientPermissions)
		default:
			log(common.LogLevelError, fmt.Sprintf("failed to verify membership for user[%s] in org[%s]: %s", session.UserId, orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester", types.ErrorDatabaseIssue)
		}
		return
	}

	_, _, canDelete, err := orgUser.CanV1(models.DatabaseConnection{Db: dbInstance}, models.ResourceOrg, models.ActionDelete)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to evaluate delete permissions for user[%s] in org[%s]: %s", session.UserId, orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester", types.ErrorDatabaseIssue)
		return
	}
	if !canDelete {
		log(common.LogLevelWarn, fmt.Sprintf("user[%s] is not authorized to delete org[%s]", session.UserId, orgId))
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "insufficient permissions", types.ErrorInsufficientPermissions)
		return
	}

	if err := orgInstance.DeleteV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to delete org[%s] by user[%s]: %s", orgId, session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to delete org", types.ErrorDatabaseIssue)
		return
	}

	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.Delete,
		ResourceId:   orgId,
		ResourceType: audit.OrgResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", DeleteOrgV1Output{IsSuccessful: true})
}

// handleCreateOrgV1 godoc
// @Summary      Creates a new organisation
// @Description  Creates a new organisation and assigns the user identified by their token as the administrator of the organisation
// @Tags         controller-service
// @Accept       json
// @Produce      json
// @Param        request body CreateOrgV1Input true "User credentials"
// @Success      200 {object} commonHttpResponse "ok"
// @Failure      403 {object} commonHttpResponse "forbidden"
// @Failure      500 {object} commonHttpResponse "internal server error"
// @Router       /api/v1/org [post]
func handleCreateOrgV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", types.ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, "successfully read body into bytes")
	var input CreateOrgV1Input
	if err := json.Unmarshal(requestBody, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", types.ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, "successfully parsed body into expected input class")

	if err := validate.OrgName(input.Name); err != nil {
		log(common.LogLevelDebug, fmt.Sprintf("user[%s] entered an invalid orgName[%s]: %s", session.UserId, input.Name, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "org name is invalid", types.ErrorInvalidInput, err.Error())
		return
	}
	if err := validate.OrgCode(input.Code); err != nil {
		log(common.LogLevelDebug, fmt.Sprintf("user[%s] entered an invalid orgCode[%s]: %s", session.UserId, input.Code, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "org code is invalid", types.ErrorInvalidInput, err.Error())
		return
	}

	// create organisation

	orgInstance, err := models.CreateOrgV1(models.CreateOrgV1Opts{
		Db:     dbInstance,
		Code:   input.Code,
		Name:   input.Name,
		Type:   models.TypeTenantOrg,
		UserId: session.UserId,
	})
	if err != nil {
		if errors.Is(err, models.ErrorDuplicateEntry) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "org already exists", types.ErrorOrgExists)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create org", types.ErrorDatabaseIssue)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("created org[%s] with id[%s]", input.Code, orgInstance.GetId()))

	// add user who is making request to the organisation as user #1

	log(common.LogLevelDebug, fmt.Sprintf("adding user[%s] to org[%s]", session.UserId, orgInstance.GetId()))
	if err := orgInstance.AddUserV1(models.AddUserToOrgV1{
		Db:         dbInstance,
		UserId:     session.UserId,
		MemberType: string(models.TypeOrgAdmin),
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to add user[%s] to org[%s]: %s", session.UserId, orgInstance.GetId(), err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to add user to org", types.ErrorDatabaseIssue)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("added user[%s] to org[%s] as admin", session.UserId, orgInstance.GetId()))

	// add a default administrator role

	log(common.LogLevelDebug, fmt.Sprintf("adding default admin role to org[%s]...", orgInstance.GetId()))
	defaultAdminRole, err := orgInstance.CreateRoleV1(models.CreateOrgRoleV1Input{
		DatabaseConnection: models.DatabaseConnection{Db: dbInstance},
		RoleName:           models.DefaultOrgRoleAdminName,
		UserId:             session.UserId,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to add default admin role to org[%s]: %s", defaultAdminRole.GetId(), err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to add role to org", types.ErrorDatabaseIssue)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("added default admin role[%s] to org[%s]", defaultAdminRole.GetId(), orgInstance.GetId()))
	log(common.LogLevelDebug, fmt.Sprintf("adding permissions to admin role[%s]...", defaultAdminRole.GetId()))
	resources := []models.Resource{
		models.ResourceAutomationLogs,
		models.ResourceAutomations,
		models.ResourceOrg,
		models.ResourceOrgBilling,
		models.ResourceOrgConfig,
		models.ResourceOrgUser,
		models.ResourceTemplates,
	}
	permissionErrs := []error{}
	for _, resource := range resources {
		if err := defaultAdminRole.CreatePermissionV1(models.CreateOrgRolePermissionV1Input{
			Allows:             models.ActionSetAdmin,
			Resource:           resource,
			DatabaseConnection: models.DatabaseConnection{Db: dbInstance},
		}); err != nil {
			permissionErrs = append(permissionErrs, err)
		}
	}
	if len(permissionErrs) > 0 {
		log(common.LogLevelError, fmt.Sprintf("failed to add permissions to admin role[%s]: %s", defaultAdminRole.GetId(), errors.Join(permissionErrs...)))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to add permissions to admin org role", types.ErrorDatabaseIssue)
		return
	}

	// create deafult role for workers to assume

	log(common.LogLevelDebug, fmt.Sprintf("adding default worker role to org[%s]...", orgInstance.GetId()))
	defaultWorkerRole, err := orgInstance.CreateRoleV1(models.CreateOrgRoleV1Input{
		DatabaseConnection: models.DatabaseConnection{Db: dbInstance},
		RoleName:           models.DefaultOrgRoleWorkerName,
		UserId:             session.UserId,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to add default worker role to org[%s]: %s", defaultWorkerRole.GetId(), err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to add role to org", types.ErrorDatabaseIssue)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("added default worker role[%s] to org[%s]", defaultWorkerRole.GetId(), orgInstance.GetId()))
	log(common.LogLevelDebug, fmt.Sprintf("adding permissions to worker orgRole[%s]...", defaultWorkerRole.GetId()))
	workerResourceActionMap := map[models.Resource]models.Action{
		models.ResourceAutomationLogs: models.ActionCreate | models.ActionUpdate | models.ActionView | models.ActionManage,
		models.ResourceAutomations:    models.ActionCreate | models.ActionUpdate | models.ActionView,
		models.ResourceTemplates:      models.ActionView,
	}
	permissionErrs = []error{}
	for resourceType, allowedActions := range workerResourceActionMap {
		if err := defaultWorkerRole.CreatePermissionV1(models.CreateOrgRolePermissionV1Input{
			Allows:             allowedActions,
			Resource:           resourceType,
			DatabaseConnection: models.DatabaseConnection{Db: dbInstance},
		}); err != nil {
			permissionErrs = append(permissionErrs, err)
		}
	}
	if len(permissionErrs) > 0 {
		log(common.LogLevelError, fmt.Sprintf("failed to add permissions to default worker orgRole[%s]: %s", defaultWorkerRole.GetId(), errors.Join(permissionErrs...)))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to add permissions to default worker org role", types.ErrorDatabaseIssue)
		return
	}

	// let the user making the request have adminsitrator permissions

	assigner := session.UserId
	if err := defaultAdminRole.AssignUserV1(models.AssignOrgRoleV1Input{
		DatabaseConnection: models.DatabaseConnection{Db: dbInstance},
		OrgId:              orgInstance.GetId(),
		UserId:             session.UserId,
		AssignedBy:         &assigner,
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to assign default administrator role[%s] to user[%s]: %s", defaultAdminRole.GetId(), session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to assign org role", types.ErrorDatabaseIssue)
		return
	}

	// provision a ca for the org for signing client certificates

	log(common.LogLevelDebug, fmt.Sprintf("creating certificate authority for org[%s]", orgInstance.GetId()))
	if _, err := orgInstance.CreateCertificateAuthorityV1(models.CreateOrgCertificateAuthorityV1Input{
		DatabaseConnection: models.DatabaseConnection{Db: dbInstance},
		CertOptions: &tls.CertificateOptions{
			NotAfter: time.Now().Add(time.Hour * 24 * 365 * 5),
		},
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to create certificate authority for org[%s]: %s", orgInstance.GetId(), err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create certificate authority", types.ErrorDatabaseIssue)
		return
	}

	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.Create,
		ResourceId:   orgInstance.GetId(),
		ResourceType: audit.OrgResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", CreateOrgV1Output{
		Id:   orgInstance.GetId(),
		Code: input.Code,
	})
}

type handleGetOrgV1Output struct {
	Code      string     `json:"code"`
	CreatedAt time.Time  `json:"createdAt"`
	Id        string     `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

func handleGetOrgV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)

	vars := mux.Vars(r)
	orgRef := vars["orgRef"]

	isorgRefUuid := false
	if err := validate.Uuid(orgRef); err == nil {
		isorgRefUuid = true
	} else if err := validate.OrgCode(orgRef); err != nil {
		log(common.LogLevelDebug, fmt.Sprintf("user[%s] entered an invalid reference[%s]: %s", session.UserId, orgRef, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "org code is invalid", types.ErrorInvalidInput)
		return
	}

	// retrieve the org
	getOrgOpts := models.GetOrgV1Opts{
		Db: dbInstance,
	}
	if isorgRefUuid {
		getOrgOpts.Id = &orgRef
	} else {
		getOrgOpts.Code = &orgRef
	}

	org, err := models.GetOrgV1(getOrgOpts)
	if err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org", types.ErrorDatabaseIssue)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org", types.ErrorDatabaseIssue)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("successfully retrieved org[%s] with reference[%s]", orgRef, org.GetId()))

	// only return the information if the user is part of the org

	orgUser, err := org.GetUserV1(models.GetOrgUserV1Opts{
		Db:     dbInstance,
		UserId: session.UserId,
	})
	if err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			log(common.LogLevelError, fmt.Sprintf("unauthorized user[%s] requested data about org[%s]: %s", session.UserId, org.GetId(), err))
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org", types.ErrorDatabaseIssue)
			return
		}
		log(common.LogLevelError, fmt.Sprintf("failed to retrieve user[%s] in org[%s]: %s", session.UserId, org.GetId(), err))
		common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org", types.ErrorDatabaseIssue)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("successfully retrieved user[%s] in org[%s]", orgUser.User.GetId(), orgUser.Org.GetId()))
	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.Get,
		ResourceId:   orgUser.User.GetId(),
		ResourceType: audit.OrgUserResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleGetOrgV1Output{
		Code:      org.Code,
		CreatedAt: org.CreatedAt,
		Id:        org.GetId(),
		Name:      org.Name,
		Type:      org.Type,
		UpdatedAt: org.UpdatedAt,
	})
}

type handleListOrgsV1Output handleListOrgsV1OutputOrgs

type handleListOrgsV1OutputOrgs []handleListOrgsV1OutputOrg
type handleListOrgsV1OutputOrg struct {
	Code       string     `json:"code"`
	CreatedAt  time.Time  `json:"createdAt"`
	Id         string     `json:"id"`
	JoinedAt   time.Time  `json:"joinedAt"`
	MemberType string     `json:"memberType"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	UpdatedAt  *time.Time `json:"updatedAt"`
}

// handleListOrgsV1 godoc
// @Summary      Retrieves the current organisation
// @Description  Retrieves the current organisation that the current user is signed in via
// @Tags         controller-service
// @Accept       json
// @Produce      json
// @Success      200 {object} commonHttpResponse "ok"
// @Failure      403 {object} commonHttpResponse "forbidden"
// @Failure      500 {object} commonHttpResponse "internal server error"
// @Router       /api/v1/org [get]
func handleListOrgsV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)

	log(common.LogLevelDebug, fmt.Sprintf("retrieving organisations that user[%s] is in", session.UserId))

	orgs, err := models.ListUserOrgsV1(models.ListUserOrgsV1Opts{
		Db:     dbInstance,
		UserId: session.UserId,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to retrieve orgs that user[%s] is in", session.UserId), err)
		return
	}

	output := handleListOrgsV1Output{}
	for _, org := range orgs {
		output = append(
			output,
			handleListOrgsV1OutputOrg{
				Code:       org.Code,
				CreatedAt:  org.CreatedAt,
				Id:         *org.Id,
				JoinedAt:   *org.JoinedAt,
				MemberType: *org.MemberType,
				Name:       org.Name,
				Type:       org.Type,
				UpdatedAt:  org.UpdatedAt,
			},
		)
	}

	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.List,
		ResourceType: audit.OrgResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}

type handleListOrgUsersV1Output []handleListOrgUsersV1OutputUser

type handleListOrgUsersV1OutputUser struct {
	JoinedAt   time.Time                            `json:"joinedAt"`
	MemberType string                               `json:"memberType"`
	OrgId      string                               `json:"orgId"`
	OrgCode    string                               `json:"orgCode"`
	OrgName    string                               `json:"orgName"`
	UserId     string                               `json:"userId"`
	UserEmail  string                               `json:"userEmail"`
	UserType   string                               `json:"userType"`
	Roles      []handleListOrgUsersV1OutputUserRole `json:"roles"`
}

type handleListOrgUsersV1OutputUserRole struct {
	CreatedAt     time.Time                                      `json:"createdAt" yaml:"createdAt"`
	CreatedBy     *handleListOrgUsersV1OutputUserRoleUser        `json:"createdBy" yaml:"createdBy"`
	Id            string                                         `json:"id" yaml:"id"`
	LastUpdatedAt time.Time                                      `json:"lastUpdatedAt" yaml:"lastUpdatedAt"`
	Name          string                                         `json:"name" yaml:"name"`
	Permissions   []handleListOrgUsersV1OutputUserRolePermission `json:"permissions" yaml:"permissions"`
}

type handleListOrgUsersV1OutputUserRoleUser struct {
	Email string `json:"email" yaml:"email"`
	Id    string `json:"id" yaml:"id"`
}

type handleListOrgUsersV1OutputUserRolePermission struct {
	Allows   uint64 `json:"allows" yaml:"allows"`
	Denys    uint64 `json:"denys" yaml:"denys"`
	Id       string `json:"id" yaml:"id"`
	Resource string `json:"resource" yaml:"resource"`
}

func handleListOrgUsersV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)
	vars := mux.Vars(r)
	orgId := vars["orgId"]

	log(common.LogLevelDebug, fmt.Sprintf("user[%s] requested list of users from org[%s]", session.UserId, orgId))
	org := models.Org{Id: &orgId}
	_, err := org.GetUserV1(models.GetOrgUserV1Opts{Db: dbInstance, UserId: session.UserId})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to get user[%s] from org[%s]: %s", session.UserId, orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, fmt.Sprintf("refused to list users in org[%s] at user[%s]'s request", orgId, session.UserId), types.ErrorInvalidInput)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org users", types.ErrorDatabaseIssue)
		return
	}
	orgUsers, err := org.ListUsersV1(models.DatabaseConnection{Db: dbInstance})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to list users from org[%s]: %s", orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org users", types.ErrorDatabaseIssue)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org users", types.ErrorDatabaseIssue)
		return
	}
	output := handleListOrgUsersV1Output{}
	for _, orgUser := range orgUsers {
		userRoles, err := orgUser.ListRolesV1(models.DatabaseConnection{Db: dbInstance})
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to list roles for user[%s] in org[%s]: %s", orgUser.User.GetId(), orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org user roles", types.ErrorDatabaseIssue)
			return
		}
		roles := make([]handleListOrgUsersV1OutputUserRole, 0, len(userRoles))
		for _, role := range userRoles {
			roleOutput := handleListOrgUsersV1OutputUserRole{
				CreatedAt:     role.CreatedAt,
				Id:            role.GetId(),
				LastUpdatedAt: role.LastUpdatedAt,
				Name:          role.Name,
				Permissions:   make([]handleListOrgUsersV1OutputUserRolePermission, 0, len(role.Permissions)),
			}
			if role.CreatedBy != nil && role.CreatedBy.Id != nil {
				roleOutput.CreatedBy = &handleListOrgUsersV1OutputUserRoleUser{
					Id:    role.CreatedBy.GetId(),
					Email: role.CreatedBy.Email,
				}
			}
			for _, permission := range role.Permissions {
				if permission.Id == nil {
					continue
				}
				roleOutput.Permissions = append(roleOutput.Permissions, handleListOrgUsersV1OutputUserRolePermission{
					Id:       *permission.Id,
					Resource: string(permission.Resource),
					Allows:   uint64(permission.Allows),
					Denys:    uint64(permission.Denys),
				})
			}
			roles = append(roles, roleOutput)
		}
		output = append(
			output,
			handleListOrgUsersV1OutputUser{
				JoinedAt:   orgUser.JoinedAt,
				MemberType: orgUser.MemberType,
				OrgId:      orgUser.Org.GetId(),
				OrgCode:    orgUser.Org.Code,
				OrgName:    orgUser.Org.Name,
				UserId:     orgUser.User.GetId(),
				UserEmail:  orgUser.User.Email,
				UserType:   string(orgUser.User.Type),
				Roles:      roles,
			},
		)
	}

	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.List,
		ResourceType: audit.OrgUserResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}

type ListOrgRolesV1Output []ListOrgRolesV1OutputRole

type ListOrgRolesV1OutputRole struct {
	CreatedAt     time.Time                            `json:"createdAt" yaml:"createdAt"`
	CreatedBy     *ListOrgRolesV1OutputRoleUser        `json:"createdBy" yaml:"createdBy"`
	Id            string                               `json:"id" yaml:"id"`
	LastUpdatedAt time.Time                            `json:"lastUpdatedAt" yaml:"lastUpdatedAt"`
	Name          string                               `json:"name" yaml:"name"`
	OrgId         string                               `json:"orgId" yaml:"orgId"`
	Permissions   []ListOrgRolesV1OutputRolePermission `json:"permissions" yaml:"permissions"`
}

type ListOrgRolesV1OutputRoleUser struct {
	Email string `json:"email" yaml:"email"`
	Id    string `json:"id" yaml:"id"`
}

type ListOrgRolesV1OutputRolePermission struct {
	Allows   uint64 `json:"allows" yaml:"allows"`
	Denys    uint64 `json:"denys" yaml:"denys"`
	Id       string `json:"id" yaml:"id"`
	Resource string `json:"resource" yaml:"resource"`
}

func handleListOrgRolesV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)
	vars := mux.Vars(r)
	orgId := vars["orgId"]

	log(common.LogLevelDebug, fmt.Sprintf("user[%s] requested list of roles from org[%s]", session.UserId, orgId))
	org := models.Org{Id: &orgId}
	if _, err := org.GetUserV1(models.GetOrgUserV1Opts{Db: dbInstance, UserId: session.UserId}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to get user[%s] from org[%s]: %s", session.UserId, orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, fmt.Sprintf("refused to list roles in org[%s] at user[%s]'s request", orgId, session.UserId), types.ErrorInvalidInput)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org roles", types.ErrorDatabaseIssue)
		return
	}
	orgRoles, err := org.ListRolesV1(models.DatabaseConnection{Db: dbInstance})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to list roles from org[%s]: %s", orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org roles", types.ErrorDatabaseIssue)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org roles", types.ErrorDatabaseIssue)
		return
	}
	output := make(ListOrgRolesV1Output, 0, len(orgRoles))
	for _, role := range orgRoles {
		var createdBy *ListOrgRolesV1OutputRoleUser
		if role.CreatedBy != nil && role.CreatedBy.Id != nil {
			createdBy = &ListOrgRolesV1OutputRoleUser{
				Id:    role.CreatedBy.GetId(),
				Email: role.CreatedBy.Email,
			}
		}
		permissions := make([]ListOrgRolesV1OutputRolePermission, 0, len(role.Permissions))
		for _, permission := range role.Permissions {
			if permission.Id == nil {
				continue
			}
			permissions = append(permissions, ListOrgRolesV1OutputRolePermission{
				Id:       *permission.Id,
				Resource: string(permission.Resource),
				Allows:   uint64(permission.Allows),
				Denys:    uint64(permission.Denys),
			})
		}
		orgIdValue := orgId
		if role.OrgId != nil {
			orgIdValue = *role.OrgId
		}
		output = append(output, ListOrgRolesV1OutputRole{
			CreatedAt:     role.CreatedAt,
			CreatedBy:     createdBy,
			Id:            role.GetId(),
			LastUpdatedAt: role.LastUpdatedAt,
			Name:          role.Name,
			OrgId:         orgIdValue,
			Permissions:   permissions,
		})
	}

	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.List,
		ResourceId:   orgId,
		ResourceType: audit.OrgResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}

func handleSubmitOrgTemplateV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)
	vars := mux.Vars(r)
	orgId := vars["orgId"]
	if orgId == "" {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "organization id was not provided", types.ErrorInvalidInput)
		return
	}
	if _, err := uuid.Parse(orgId); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "organization id is not a valid uuid", types.ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("user[%s] is creating an automation template for org[%s]", session.UserId, orgId))

	org := models.Org{Id: &orgId}
	if _, err := org.GetUserV1(models.GetOrgUserV1Opts{
		Db:     dbInstance,
		UserId: session.UserId,
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to verify membership of user[%s] in org[%s]: %s", session.UserId, orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusForbidden, "user is not part of organization", types.ErrorInsufficientPermissions)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "organization membership verification failed", types.ErrorDatabaseIssue)
		return
	}

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

	automationTemplateVersion, err := models.SubmitOrgTemplateV1(models.SubmitOrgTemplateV1Opts{
		Db:       dbInstance,
		OrgId:    orgId,
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

func handleListOrgTemplatesV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)
	vars := mux.Vars(r)
	orgId := vars["orgId"]
	if orgId == "" {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "organization id was not provided", types.ErrorInvalidInput)
		return
	}
	if _, err := uuid.Parse(orgId); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "organization id is not a valid uuid", types.ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("retrieving automation templates for org[%s] by user[%s]", orgId, session.UserId))

	org := models.Org{Id: &orgId}
	if _, err := org.GetUserV1(models.GetOrgUserV1Opts{
		Db:     dbInstance,
		UserId: session.UserId,
	}); err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusForbidden, "user is not authorized to access organization templates", types.ErrorInsufficientPermissions)
			return
		}
		log(common.LogLevelError, fmt.Sprintf("failed to verify membership of user[%s] in org[%s]: %s", session.UserId, orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "organization membership verification failed", types.ErrorDatabaseIssue)
		return
	}

	templates, err := org.ListTemplatesV1(models.DatabaseConnection{Db: dbInstance})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to list org templates for org[%s]: %s", orgId, err))
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

type ListOrgTokensV1Output []ListOrgTokensV1OutputToken

type ListOrgTokensV1OutputToken struct {
	Id            string                          `json:"id" yaml:"id"`
	OrgId         string                          `json:"orgId" yaml:"orgId"`
	Name          string                          `json:"name" yaml:"name"`
	Description   *string                         `json:"description" yaml:"description"`
	CreatedAt     time.Time                       `json:"createdAt" yaml:"createdAt"`
	CreatedBy     *ListOrgTokensV1OutputTokenUser `json:"createdBy" yaml:"createdBy"`
	LastUpdatedAt time.Time                       `json:"lastUpdatedAt" yaml:"lastUpdatedAt"`
	LastUpdatedBy *ListOrgTokensV1OutputTokenUser `json:"lastUpdatedBy" yaml:"lastUpdatedBy"`
}

type ListOrgTokensV1OutputTokenUser struct {
	Id    string `json:"id" yaml:"id"`
	Email string `json:"email" yaml:"email"`
}

func handleListOrgTokensV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)
	vars := mux.Vars(r)
	orgId := vars["orgId"]

	log(common.LogLevelDebug, fmt.Sprintf("user[%s] requested list of tokens from org[%s]", session.UserId, orgId))

	if err := validateRequesterCanManageOrgUsers(validateRequesterCanManageOrgUsersOpts{
		OrgId:           orgId,
		RequesterUserId: session.UserId,
	}); err != nil {
		switch {
		case errors.Is(err, types.ErrorInsufficientPermissions):
			log(common.LogLevelError, fmt.Sprintf("user[%s] is not authorized to list tokens for org[%s]", session.UserId, orgId))
			common.SendHttpFailResponse(w, r, http.StatusForbidden, "requester is not authorized to list tokens", types.ErrorInsufficientPermissions)
			return
		case errors.Is(err, types.ErrorDatabaseIssue):
			log(common.LogLevelError, fmt.Sprintf("failed to verify requester permissions for org[%s]: %s", orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester permissions", types.ErrorDatabaseIssue)
			return
		default:
			log(common.LogLevelError, fmt.Sprintf("unexpected error verifying requester permissions for org[%s]: %s", orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester permissions", types.ErrorUnknown)
			return
		}
	}

	org := models.Org{Id: &orgId}
	orgTokens, err := org.ListTokensV1(models.ListOrgTokensV1Opts{DatabaseConnection: models.DatabaseConnection{Db: dbInstance}})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to list tokens for org[%s]: %s", orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org tokens", types.ErrorDatabaseIssue)
		return
	}

	redactedTokens := orgTokens.GetRedacted()
	output := make(ListOrgTokensV1Output, 0, len(redactedTokens))
	for _, token := range redactedTokens {
		var createdBy *ListOrgTokensV1OutputTokenUser
		if token.CreatedBy != nil && token.CreatedBy.Id != nil {
			createdBy = &ListOrgTokensV1OutputTokenUser{
				Id:    *token.CreatedBy.Id,
				Email: token.CreatedBy.Email,
			}
		}
		var lastUpdatedBy *ListOrgTokensV1OutputTokenUser
		if token.LastUpdatedBy != nil && token.LastUpdatedBy.Id != nil {
			lastUpdatedBy = &ListOrgTokensV1OutputTokenUser{
				Id:    *token.LastUpdatedBy.Id,
				Email: token.LastUpdatedBy.Email,
			}
		}
		output = append(output, ListOrgTokensV1OutputToken{
			Id:            token.GetId(),
			OrgId:         token.GetOrg().GetId(),
			Name:          token.Name,
			Description:   token.Description,
			CreatedAt:     token.CreatedAt,
			CreatedBy:     createdBy,
			LastUpdatedAt: token.LastUpdatedAt,
			LastUpdatedBy: lastUpdatedBy,
		})
	}

	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.List,
		ResourceId:   orgId,
		ResourceType: audit.OrgResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}

type GetOrgTokenV1Output struct {
	Id            string                          `json:"id" yaml:"id"`
	OrgId         string                          `json:"orgId" yaml:"orgId"`
	Name          string                          `json:"name" yaml:"name"`
	Description   *string                         `json:"description" yaml:"description"`
	CreatedAt     time.Time                       `json:"createdAt" yaml:"createdAt"`
	CreatedBy     *ListOrgTokensV1OutputTokenUser `json:"createdBy" yaml:"createdBy"`
	LastUpdatedAt time.Time                       `json:"lastUpdatedAt" yaml:"lastUpdatedAt"`
	LastUpdatedBy *ListOrgTokensV1OutputTokenUser `json:"lastUpdatedBy" yaml:"lastUpdatedBy"`
	Role          *GetOrgTokenV1OutputRole        `json:"role" yaml:"role"`
}

type GetOrgTokenV1OutputRole struct {
	Id          string                              `json:"id" yaml:"id"`
	Name        string                              `json:"name" yaml:"name"`
	Permissions []GetOrgTokenV1OutputRolePermission `json:"permissions" yaml:"permissions"`
}

type GetOrgTokenV1OutputRolePermission struct {
	Id       string `json:"id" yaml:"id"`
	Resource string `json:"resource" yaml:"resource"`
	Allows   uint64 `json:"allows" yaml:"allows"`
	Denys    uint64 `json:"denys" yaml:"denys"`
}

func handleGetOrgTokenV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)
	vars := mux.Vars(r)
	orgId := vars["orgId"]
	tokenId := vars["tokenId"]

	if err := validate.Uuid(tokenId); err != nil {
		log(common.LogLevelDebug, fmt.Sprintf("user[%s] provided invalid tokenId[%s]: %s", session.UserId, tokenId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "token id is invalid", types.ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("user[%s] requested token[%s] from org[%s]", session.UserId, tokenId, orgId))

	if err := validateRequesterCanManageOrgUsers(validateRequesterCanManageOrgUsersOpts{
		OrgId:           orgId,
		RequesterUserId: session.UserId,
	}); err != nil {
		switch {
		case errors.Is(err, types.ErrorInsufficientPermissions):
			log(common.LogLevelError, fmt.Sprintf("user[%s] is not authorized to view token[%s] for org[%s]", session.UserId, tokenId, orgId))
			common.SendHttpFailResponse(w, r, http.StatusForbidden, "requester is not authorized to view token", types.ErrorInsufficientPermissions)
			return
		case errors.Is(err, types.ErrorDatabaseIssue):
			log(common.LogLevelError, fmt.Sprintf("failed to verify requester permissions for org[%s]: %s", orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester permissions", types.ErrorDatabaseIssue)
			return
		default:
			log(common.LogLevelError, fmt.Sprintf("unexpected error verifying requester permissions for org[%s]: %s", orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester permissions", types.ErrorUnknown)
			return
		}
	}

	org := models.Org{Id: &orgId}
	orgToken, err := org.GetTokenByIdV1(models.GetOrgTokenByIdV1Opts{
		DatabaseConnection: models.DatabaseConnection{Db: dbInstance},
		TokenId:            tokenId,
	})
	if err != nil {
		switch {
		case errors.Is(err, models.ErrorNotFound):
			log(common.LogLevelDebug, fmt.Sprintf("token[%s] in org[%s] not found", tokenId, orgId))
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "token was not found", types.ErrorNotFound)
			return
		default:
			log(common.LogLevelError, fmt.Sprintf("failed to load token[%s] for org[%s]: %s", tokenId, orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org token", types.ErrorDatabaseIssue)
			return
		}
	}

	redactedToken := orgToken.GetRedacted()
	var createdBy *ListOrgTokensV1OutputTokenUser
	if redactedToken.CreatedBy != nil && redactedToken.CreatedBy.Id != nil {
		createdBy = &ListOrgTokensV1OutputTokenUser{Id: *redactedToken.CreatedBy.Id}
	}
	var lastUpdatedBy *ListOrgTokensV1OutputTokenUser
	if redactedToken.LastUpdatedBy != nil && redactedToken.LastUpdatedBy.Id != nil {
		lastUpdatedBy = &ListOrgTokensV1OutputTokenUser{Id: *redactedToken.LastUpdatedBy.Id}
	}

	var roleOutput *GetOrgTokenV1OutputRole
	if redactedToken.Role != nil {
		permissions := make([]GetOrgTokenV1OutputRolePermission, 0, len(redactedToken.Role.Permissions))
		for _, permission := range redactedToken.Role.Permissions {
			permissionId := ""
			if permission.Id != nil {
				permissionId = *permission.Id
			}
			permissions = append(permissions, GetOrgTokenV1OutputRolePermission{
				Id:       permissionId,
				Resource: string(permission.Resource),
				Allows:   uint64(permission.Allows),
				Denys:    uint64(permission.Denys),
			})
		}
		roleOutput = &GetOrgTokenV1OutputRole{
			Id:          redactedToken.Role.GetId(),
			Name:        redactedToken.Role.Name,
			Permissions: permissions,
		}
	}

	output := GetOrgTokenV1Output{
		Id:            redactedToken.GetId(),
		OrgId:         redactedToken.GetOrg().GetId(),
		Name:          redactedToken.Name,
		Description:   redactedToken.Description,
		CreatedAt:     redactedToken.CreatedAt,
		CreatedBy:     createdBy,
		LastUpdatedAt: redactedToken.LastUpdatedAt,
		LastUpdatedBy: lastUpdatedBy,
		Role:          roleOutput,
	}

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}

type CreateOrgTokenV1Input struct {
	Description *string `json:"description"`
	Name        string  `json:"name"`
	RoleId      string  `json:"roleId"`
	OrgId       string  `json:"-"`
}

type CreateOrgTokenV1Output struct {
	TokenId           string `json:"tokenId"`
	Name              string `json:"name"`
	ApiKey            string `json:"apiKey"`
	CertificatePem    string `json:"certificatePem"`
	CertificateBase64 string `json:"certificateB64"`
	PrivateKeyPem     string `json:"privateKeyPem"`
	PrivateKeyBase64  string `json:"privateKeyB64"`
}

func handleCreateOrgTokenV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)
	vars := mux.Vars(r)
	orgId := vars["orgId"]

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", types.ErrorInvalidInput)
		return
	}
	var input CreateOrgTokenV1Input
	if err := json.Unmarshal(requestBody, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", types.ErrorInvalidInput)
		return
	}
	if input.Name == "" {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "token name is required", types.ErrorInvalidInput)
		return
	}
	if input.RoleId == "" {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "role id is required", types.ErrorInvalidInput)
		return
	}
	if err := validateRequesterCanManageOrgUsers(validateRequesterCanManageOrgUsersOpts{
		OrgId:           orgId,
		RequesterUserId: session.UserId,
	}); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "requester is not authorized to manage tokens", types.ErrorInsufficientPermissions)
		return
	}

	org := models.Org{Id: &orgId}
	orgDetails, err := models.GetOrgV1(models.GetOrgV1Opts{Db: dbInstance, Id: &orgId})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to get org[%s]: %s", orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org", types.ErrorNotFound)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org", types.ErrorDatabaseIssue)
		return
	}

	orgRole, err := org.GetRoleByIdV1(models.GetOrgRoleByIdV1Opts{
		DatabaseConnection: models.DatabaseConnection{Db: dbInstance},
		RoleId:             input.RoleId,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to load role[%s] for org[%s]: %s", input.RoleId, orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "specified role was not found", types.ErrorNotFound)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to load org role", types.ErrorDatabaseIssue)
		return
	}

	ca, err := org.LoadCertificateAuthorityV1(models.LoadOrgCertificateAuthorityV1Opts{
		DatabaseConnection: models.DatabaseConnection{Db: dbInstance},
	})
	if err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			log(common.LogLevelDebug, fmt.Sprintf("no certificate authority for org[%s], creating new one", orgId))
			ca, err = org.CreateCertificateAuthorityV1(models.CreateOrgCertificateAuthorityV1Input{
				DatabaseConnection: models.DatabaseConnection{Db: dbInstance},
			})
		}
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to prepare certificate authority for org[%s]: %s", orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to prepare certificate authority", types.ErrorDatabaseIssue)
			return
		}
	}

	caCert, caKey, err := ca.GetCryptoMaterials()
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to parse certificate authority for org[%s]: %s", orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to load certificate authority", types.ErrorDatabaseIssue)
		return
	}

	apiKey, err := generateApiKey(constants.ApiKeyLength - len(constants.ApiKeyPrefix))
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to generate api key for org[%s]: %s", orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to generate api key", types.ErrorUnknown)
		return
	}
	apiKey = constants.ApiKeyPrefix + apiKey

	tokenId := uuid.NewString()
	certOpts := tls.CertificateOptions{
		CommonName:         orgDetails.Code,
		Organization:       []string{orgDetails.GetId()},
		OrganizationalUnit: []string{tokenId, apiKey},
		IsClient:           true,
	}
	certOpts.NotAfter = caCert.NotAfter
	if certOpts.NotAfter.After(caCert.NotAfter) {
		certOpts.NotAfter = caCert.NotAfter
	}
	certificate, key, err := tls.GenerateCertificate(&certOpts, caCert, caKey)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to generate certificate for org[%s]: %s", orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to generate certificate", types.ErrorUnknown)
		return
	}

	createdBy := session.UserId
	orgToken, err := org.CreateTokenV1(models.CreateOrgTokenV1Input{
		TokenId:            tokenId,
		DatabaseConnection: models.DatabaseConnection{Db: dbInstance},
		Name:               input.Name,
		Description:        input.Description,
		CertificatePem:     certificate.Pem,
		PrivateKeyPem:      key.Pem,
		ApiKey:             apiKey,
		CreatedBy:          &createdBy,
		OrgRole:            orgRole,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to create org token for org[%s]: %s", orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create org token", types.ErrorDatabaseIssue)
		return
	}

	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.Create,
		ResourceId:   orgToken.GetId(),
		ResourceType: audit.OrgResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})

	output := CreateOrgTokenV1Output{
		TokenId:           orgToken.GetId(),
		Name:              orgToken.Name,
		ApiKey:            apiKey,
		CertificateBase64: base64.StdEncoding.EncodeToString(certificate.Pem),
		CertificatePem:    string(certificate.Pem),
		PrivateKeyBase64:  base64.StdEncoding.EncodeToString(key.Pem),
		PrivateKeyPem:     string(key.Pem),
	}
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}

type handleCreateOrgUserV1Input struct {
	Email                 string `json:"email"`
	Type                  string `json:"type"`
	IsTriggerEmailEnabled bool   `json:"isTriggerEmailEnabled"`
}
type handleCreateOrgUserV1Output struct {
	Id             string `json:"id"`
	JoinCode       string `json:"joinCode"`
	IsExistingUser bool   `json:"isExistingUser"`
}

func handleCreateOrgUserV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)

	vars := mux.Vars(r)
	orgId := vars["orgId"]

	log(common.LogLevelDebug, fmt.Sprintf("user[%s] is adding another user to org[%s]", session.UserId, orgId))

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", types.ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, "successfully read body into bytes")
	var input handleCreateOrgUserV1Input
	if err := json.Unmarshal(requestBody, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", types.ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, "successfully parsed body into expected input class")

	if err := validate.Email(input.Email); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to validate invitee's email: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to receive valid email", types.ErrorInvalidInput)
		return
	}

	org, err := models.GetOrgV1(models.GetOrgV1Opts{
		Db: dbInstance,
		Id: &orgId,
	})
	if err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to validate org", types.ErrorInvalidInput)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org", types.ErrorDatabaseIssue)
		return
	}

	if err := validateRequesterCanManageOrgUsers(validateRequesterCanManageOrgUsersOpts{
		OrgId:           orgId,
		RequesterUserId: session.UserId,
	}); err != nil {
		switch true {
		case errors.Is(err, types.ErrorInsufficientPermissions):
			log(common.LogLevelError, fmt.Sprintf("user[%s] doesn't have permissions to add a member to org[%s]: %s", session.UserId, orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to verify requester", types.ErrorInsufficientPermissions)
			return
		case errors.Is(err, types.ErrorDatabaseIssue):
			log(common.LogLevelError, fmt.Sprintf("encountered database issue while processing request by user[%s] to add user with email[%s] to org[%s]: %s", session.UserId, input.Email, orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester", types.ErrorDatabaseIssue)
			return
		}
	}

	isAcceptorExists := false
	acceptor := models.User{Email: input.Email}
	if err := acceptor.LoadByEmailV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		if !errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve acceptor", types.ErrorDatabaseIssue)
			return
		}
	} else {
		isAcceptorExists = true
	}

	joinCode, err := common.GenerateRandomString(32)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to generate random string: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to generate join code", err)
		return
	}
	invitationOpts := models.InviteOrgUserV1Opts{
		Db:             dbInstance,
		InviterId:      session.UserId,
		JoinCode:       joinCode,
		MembershipType: input.Type,
	}
	if isAcceptorExists {
		invitationOpts.AcceptorId = acceptor.Id
		if _, err = org.GetUserV1(models.GetOrgUserV1Opts{
			Db:     dbInstance,
			UserId: session.UserId,
		}); err != nil {
			if !errors.Is(err, models.ErrorNotFound) {
				common.SendHttpFailResponse(w, r, http.StatusBadRequest, "user already in org", types.ErrorUserExistsInOrg)
				return
			}
		}
	} else {
		invitationOpts.AcceptorEmail = &input.Email
	}

	invitationOutput, err := org.InviteUserV1(invitationOpts)
	if err != nil {
		if errors.Is(err, models.ErrorDuplicateEntry) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invitation exists", types.ErrorInvitationExists)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create invitation", err)
		return
	}

	if input.IsTriggerEmailEnabled {
		// TODO: send notification email
		if smtpConfig.IsSet() {
			remoteAddr := r.RemoteAddr
			userAgent := r.UserAgent()
			opsicleCatMimeType, opsicleCatData := images.GetOpsicleCat()
			if err := email.SendSmtp(email.SendSmtpOpts{
				ServiceLogs: *serviceLogs,
				To: []email.User{
					{
						Address: input.Email,
					},
				},
				Sender: smtpConfig.Sender,
				Message: email.Message{
					Title: fmt.Sprintf("You have been invited to %s on Opsicle!", org.Name),
					Body: templates.GetOrgInviteNotificationMessage(
						publicServerUrl.String(),
						joinCode,
						remoteAddr,
						userAgent,
						session.Username,
						org.Name,
						org.Code,
					),
					Images: map[string]email.MessageAttachment{
						"cat.png": {
							Type: opsicleCatMimeType,
							Data: opsicleCatData,
						},
					},
				},
				Smtp: email.SmtpConfig{
					Hostname: smtpConfig.Hostname,
					Port:     smtpConfig.Port,
					Username: smtpConfig.Username,
					Password: smtpConfig.Password,
				},
			}); err != nil {
				log(common.LogLevelWarn, fmt.Sprintf("failed to send email, send user their join code[%s] manually", joinCode))
			}
		} else {
			log(common.LogLevelWarn, fmt.Sprintf("smtp is not available, send user their join code[%s] manually", joinCode))
		}
	}

	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.Create,
		ResourceId:   invitationOutput.InvitationId,
		ResourceType: audit.OrgUserInvitationResource,
		FieldId:      acceptor.Id,
		FieldType:    audit.UserResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})

	output := handleCreateOrgUserV1Output{
		Id:             invitationOutput.InvitationId,
		JoinCode:       joinCode,
		IsExistingUser: invitationOutput.IsExistingUser,
	}
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}

type handleGetOrgCurrentUserV1Output struct {
	JoinedAt   time.Time `json:"joinedAt"`
	MemberType string    `json:"memberType"`
	OrgCode    string    `json:"orgCode"`
	OrgId      string    `json:"orgId"`
	UserId     string    `json:"userId"`

	Permissions OrgUserMemberPermissions `json:"permissions"`
}

func handleGetOrgCurrentUserV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)

	vars := mux.Vars(r)
	orgId := vars["orgId"]
	if _, err := uuid.Parse(orgId); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid org id: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid org id", types.ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("received request by user[%s] to get information on their membership in org[%s]", session.UserId, orgId))

	org := models.Org{Id: &orgId}
	orgUser, err := org.GetUserV1(models.GetOrgUserV1Opts{
		Db:     dbInstance,
		UserId: session.UserId,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to retrieve user[%s] from org[%s]: %s", session.UserId, orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get user", types.ErrorInvalidInput)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to get user", types.ErrorDatabaseIssue)
		return
	}
	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.Create,
		ResourceId:   session.UserId,
		ResourceType: audit.OrgUserResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleGetOrgCurrentUserV1Output{
		JoinedAt:   orgUser.JoinedAt,
		MemberType: orgUser.MemberType,
		OrgCode:    orgUser.Org.Code,
		OrgId:      orgUser.Org.GetId(),
		UserId:     orgUser.User.GetId(),

		Permissions: OrgUserMemberPermissions{
			CanManageUsers: isAllowedToManageOrgUsers(orgUser),
		},
	})
}

type handleUpdateOrgInvitationV1Output struct {
	JoinedAt       time.Time `json:"joinedAt"`
	MembershipType string    `json:"membershipType"`
	OrgId          string    `json:"orgId"`
	OrgCode        string    `json:"orgCode"`
	OrgName        string    `json:"orgName"`
	UserId         string    `json:"userId"`
}

type handleUpdateOrgInvitationV1Input struct {
	IsAcceptance bool   `json:"isAcceptance"`
	JoinCode     string `json:"joinCode"`
}

func handleUpdateOrgInvitationV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)
	vars := mux.Vars(r)
	invitationId := vars["invitationId"]

	log(common.LogLevelDebug, fmt.Sprintf("updating user[%s]'s organisation invite[%s]", session.UserId, invitationId))

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", types.ErrorInvalidInput)
		return
	}

	var input handleUpdateOrgInvitationV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", types.ErrorInvalidInput)
		return
	}

	orgInvite := models.OrgUserInvitation{Id: invitationId}
	if err := orgInvite.LoadV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed find invitation", types.ErrorInvitationInvalid)
			return
		}
		log(common.LogLevelError, fmt.Sprintf("failed to load invitation: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to load", types.ErrorInvalidInput)
		return
	}

	if !strings.EqualFold(*orgInvite.AcceptorId, session.UserId) {
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "user not allowed", types.ErrorInvitationInvalid)
		return
	}
	if strings.Compare(orgInvite.JoinCode, input.JoinCode) != 0 {
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "invalid join code", types.ErrorInvitationInvalid)
		return
	}

	if input.IsAcceptance {
		org := models.Org{Id: &orgInvite.OrgId}
		if err := org.AddUserV1(models.AddUserToOrgV1{
			Db:         dbInstance,
			UserId:     session.UserId,
			MemberType: orgInvite.Type,
		}); err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to add user to org: %s", err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to add user", types.ErrorDatabaseIssue)
			return
		}

		if err := orgInvite.DeleteByIdV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to delete invitation", types.ErrorDatabaseIssue)
			return
		}

		orgUser, err := org.GetUserV1(models.GetOrgUserV1Opts{
			Db:     dbInstance,
			UserId: session.UserId,
		})
		if err != nil {
			if errors.Is(err, models.ErrorNotFound) {
				common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org user", types.ErrorDatabaseIssue)
				return
			}
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org user", types.ErrorDatabaseIssue)
			return
		}

		output := handleUpdateOrgInvitationV1Output{
			JoinedAt:       orgUser.JoinedAt,
			MembershipType: orgUser.MemberType,
			OrgId:          orgUser.Org.GetId(),
			OrgCode:        orgUser.Org.Code,
			OrgName:        orgUser.Org.Name,
			UserId:         orgUser.User.GetId(),
		}
		audit.Log(audit.LogEntry{
			EntityId:     session.UserId,
			EntityType:   audit.UserEntity,
			Verb:         audit.Update,
			ResourceId:   orgInvite.Id,
			ResourceType: audit.OrgUserInvitationResource,
			Status:       audit.Success,
			SrcIp:        &session.SourceIp,
			SrcUa:        &session.UserAgent,
			DstHost:      &r.Host,
		})
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
	} else {
		if err := orgInvite.DeleteByIdV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to delete invitation", types.ErrorDatabaseIssue)
			return
		}
		audit.Log(audit.LogEntry{
			EntityId:     session.UserId,
			EntityType:   audit.UserEntity,
			Verb:         audit.Delete,
			ResourceId:   orgInvite.Id,
			ResourceType: audit.OrgUserInvitationResource,
			Status:       audit.Success,
			SrcIp:        &session.SourceIp,
			SrcUa:        &session.UserAgent,
			DstHost:      &r.Host,
		})
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", nil)
	}
}

type handleLeaveOrgV1Output struct {
	IsSuccessful bool `json:"isSuccessful"`
}

func handleLeaveOrgV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)

	vars := mux.Vars(r)
	orgId := vars["orgId"]
	if _, err := uuid.Parse(orgId); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid org id: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid org id", types.ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("received request by user[%s] to leave org[%s]", session.UserId, orgId))

	orgUser := models.NewOrgUser()
	orgUser.Org.Id = &orgId
	orgUser.User.Id = &session.UserId
	if err := orgUser.LoadV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to load user to be removed", types.ErrorDatabaseIssue)
		return
	}
	if err := validateUserIsNotLastAdmin(validateUserIsNotLastAdminOpts{
		OrgId:  orgId,
		UserId: session.UserId,
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] is the last admin and cannot leave the organisation: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "user is last admin", types.ErrorLastOrgAdmin)
		return
	}

	// delete the org user

	if err := orgUser.DeleteV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to remove user[%s] from org[%s]: %s", session.UserId, orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to remove member", types.ErrorDatabaseIssue)
		return
	}

	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.Delete,
		ResourceId:   session.UserId,
		ResourceType: audit.OrgUserResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleLeaveOrgV1Output{IsSuccessful: true})
}

type handleDeleteOrgUserV1Output struct {
	IsSuccessful bool `json:"isSuccessful"`
}

func handleDeleteOrgUserV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)

	vars := mux.Vars(r)
	orgId := vars["orgId"]
	if _, err := uuid.Parse(orgId); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid org id: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid org id", types.ErrorInvalidInput)
		return
	}

	userId := vars["userId"]
	if _, err := uuid.Parse(userId); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid user id: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid user id", types.ErrorInvalidInput)
		return
	}
	if userId == session.UserId {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "a user cannot delete itself", types.ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("received request by user[%s] to delete user[%s] from org[%s]", session.UserId, userId, orgId))

	// verify requester can update an org user

	if err := validateRequesterCanManageOrgUsers(validateRequesterCanManageOrgUsersOpts{
		OrgId:           orgId,
		RequesterUserId: session.UserId,
	}); err != nil {
		switch true {
		case errors.Is(err, types.ErrorInsufficientPermissions):
			log(common.LogLevelError, fmt.Sprintf("user[%s] doesn't have permissions to delete user[%s] from org[%s]: %s", session.UserId, userId, orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to verify requester", types.ErrorInsufficientPermissions)
			return
		case errors.Is(err, types.ErrorDatabaseIssue):
			log(common.LogLevelError, fmt.Sprintf("encountered database issue while processing request by user[%s] to delete user[%s] from org[%s]: %s", session.UserId, userId, orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester", types.ErrorDatabaseIssue)
			return
		}
	}
	log(common.LogLevelDebug, fmt.Sprintf("validated user[%s] is able to manage org[%s] members", session.UserId, orgId))

	orgUser := models.NewOrgUser()
	orgUser.Org.Id = &orgId
	orgUser.User.Id = &session.UserId
	if err := orgUser.LoadV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to load user to be removed", types.ErrorDatabaseIssue)
		return
	}
	if err := validateUserIsNotLastAdmin(validateUserIsNotLastAdminOpts{
		OrgId:  orgId,
		UserId: session.UserId,
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] is the last admin and cannot leave the organisation: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "user is last admin", types.ErrorLastOrgAdmin)
		return
	}

	// delete the org user

	if err := orgUser.DeleteV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to remove user[%s] from org[%s]: %s", userId, orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to remove member", types.ErrorDatabaseIssue)
		return
	}

	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.Delete,
		ResourceId:   userId,
		ResourceType: audit.OrgUserResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleDeleteOrgUserV1Output{IsSuccessful: true})
}

type CanOrgUserActionV1OutputData struct {
	Action    string `json:"action"`
	Allows    uint64 `json:"allows"`
	Denys     uint64 `json:"denys"`
	IsAllowed bool   `json:"isAllowed"`
	OrgId     string `json:"orgId"`
	Resource  string `json:"resource"`
	UserId    string `json:"userId"`
}

func handleCanOrgUserActionV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)
	vars := mux.Vars(r)

	orgId := vars["orgId"]
	userId := vars["userId"]
	actionRaw := vars["action"]
	resourceRaw := vars["resource"]

	if err := validate.Uuid(orgId); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid org id", types.ErrorInvalidInput)
		return
	}
	if err := validate.Uuid(userId); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid user id", types.ErrorInvalidInput)
		return
	}

	actionValue, actionCanonical, err := mapOrgPermissionAction(actionRaw)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] provided invalid action[%s]: %s", session.UserId, actionRaw, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid action", types.ErrorInvalidInput)
		return
	}
	resourceValue, resourceCanonical, err := mapOrgPermissionResource(resourceRaw)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] provided invalid resource[%s]: %s", session.UserId, resourceRaw, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid resource", types.ErrorInvalidInput)
		return
	}

	org := models.Org{Id: &orgId}
	orgUser, err := org.GetUserV1(models.GetOrgUserV1Opts{Db: dbInstance, UserId: userId})
	if err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			log(common.LogLevelWarn, fmt.Sprintf("user[%s] requested permissions for user[%s] not found in org[%s]", session.UserId, userId, orgId))
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "user not found", types.ErrorNotFound)
			return
		}
		log(common.LogLevelError, fmt.Sprintf("failed to load org user[%s] in org[%s]: %s", userId, orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to load user", types.ErrorDatabaseIssue)
		return
	}

	allowsMask, denysMask, isAllowed, err := orgUser.CanV1(models.DatabaseConnection{Db: dbInstance}, resourceValue, actionValue)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed evaluating permissions for user[%s] in org[%s] action[%s] resource[%s]: %s", userId, orgId, actionCanonical, resourceCanonical, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to evaluate permissions", types.ErrorDatabaseIssue)
		return
	}

	output := CanOrgUserActionV1OutputData{
		Action:    actionCanonical,
		Allows:    uint64(allowsMask),
		Denys:     uint64(denysMask),
		IsAllowed: isAllowed,
		OrgId:     orgId,
		Resource:  resourceCanonical,
		UserId:    userId,
	}

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}

type handleUpdateOrgUserV1Output struct {
	IsSuccessful bool `json:"isSuccessful"`
}

type handleUpdateOrgUserV1Input struct {
	Update map[string]any `json:"update"`
	User   string         `json:"user"`
}

func handleUpdateOrgUserV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)

	vars := mux.Vars(r)
	orgId := vars["orgId"]
	if _, err := uuid.Parse(orgId); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid org id: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid org id", types.ErrorInvalidInput)
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", types.ErrorInvalidInput)
		return
	}

	var input handleUpdateOrgUserV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", types.ErrorInvalidInput)
		return
	}

	isUserEmail := false
	if _, err := uuid.Parse(input.User); err != nil {
		if err := validate.Email(input.User); err != nil {
			log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid user id: %s", session.UserId, err))
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid user id", types.ErrorInvalidInput)
			return
		} else {
			isUserEmail = true
		}
	}

	userId := ""

	if isUserEmail {
		user := models.User{Email: input.User}
		if err := user.LoadByEmailV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
			log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid user email: %s", session.UserId, err))
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid user identifier", types.ErrorInvalidInput)
			return
		}
		userId = user.GetId()
	} else {
		userId = input.User
	}

	// validate input

	// verify requester can update an org user

	if err := validateRequesterCanManageOrgUsers(validateRequesterCanManageOrgUsersOpts{
		OrgId:           orgId,
		RequesterUserId: session.UserId,
	}); err != nil {
		switch true {
		case errors.Is(err, types.ErrorInsufficientPermissions):
			common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to verify requester", types.ErrorInsufficientPermissions)
			return
		case errors.Is(err, types.ErrorDatabaseIssue):
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester", types.ErrorDatabaseIssue)
			return
		}
	}

	orgUser := models.NewOrgUser()
	orgUser.Org.Id = &orgId
	orgUser.User.Id = &session.UserId
	if err := orgUser.LoadV1(models.DatabaseConnection{Db: dbInstance}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to load user[%s] in org[%s]: %s", userId, orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org user", types.ErrorDatabaseIssue)
		return
	}

	// if changing permissions, ensure the only administrator is not updated to
	// remove administrator permissions if they're the only administrator
	// remaining

	if orgUser.MemberType == string(models.TypeOrgAdmin) {
		if membershipType, ok := input.Update["type"]; ok {
			if membershipType.(string) != string(models.TypeOrgAdmin) {
				org := models.Org{Id: &orgId}
				adminsCount, err := org.GetRoleCountV1(models.GetRoleCountV1Opts{
					Db:   dbInstance,
					Role: models.TypeOrgAdmin,
				})
				if err != nil {
					log(common.LogLevelError, fmt.Sprintf("encountered database issue while retrieving member count from org[%s]: %s", orgId, err))
					common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester", types.ErrorDatabaseIssue)
					return
				}
				if adminsCount == 1 {
					common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to update membership type of the only administrator", types.ErrorInvalidInput)
					return
				}
			}
		}
	}

	// make org user changes

	if err := orgUser.UpdateFieldsV1(models.UpdateFieldsV1{
		Db:          dbInstance,
		FieldsToSet: input.Update,
	}); err != nil {
		if errors.Is(err, models.ErrorInvalidInput) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to update org user", types.ErrorInvalidInput)
			return
		}
		log(common.LogLevelError, fmt.Sprintf("request by user[%s] to update user[%s] in org[%s] failed: %s", session.UserId, userId, orgId, err))
		if errors.Is(err, models.ErrorInvalidInput) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to update org user", types.ErrorInvalidInput)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to update org user", types.ErrorDatabaseIssue)
		return
	}

	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.Delete,
		ResourceId:   userId,
		ResourceType: audit.OrgUserResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleUpdateOrgUserV1Output{IsSuccessful: true})
}

func mapOrgPermissionAction(action string) (models.Action, string, error) {
	value := strings.ToLower(strings.TrimSpace(action))
	switch value {
	case "create":
		return models.ActionCreate, "create", nil
	case "view", "read", "get":
		return models.ActionView, "view", nil
	case "update", "patch", "put":
		return models.ActionUpdate, "update", nil
	case "delete", "remove":
		return models.ActionDelete, "delete", nil
	case "execute", "run":
		return models.ActionExecute, "execute", nil
	case "manage", "admin":
		return models.ActionManage, "manage", nil
	default:
		return 0, "", fmt.Errorf("unsupported action: %s", action)
	}
}

func mapOrgPermissionResource(resource string) (models.Resource, string, error) {
	value := strings.ToLower(strings.TrimSpace(resource))
	value = strings.ReplaceAll(value, "-", "_")
	switch value {
	case string(models.ResourceTemplates):
		return models.ResourceTemplates, value, nil
	case string(models.ResourceAutomations):
		return models.ResourceAutomations, value, nil
	case string(models.ResourceAutomationLogs):
		return models.ResourceAutomationLogs, value, nil
	case string(models.ResourceOrg):
		return models.ResourceOrg, value, nil
	case string(models.ResourceOrgBilling):
		return models.ResourceOrgBilling, value, nil
	case string(models.ResourceOrgConfig):
		return models.ResourceOrgConfig, value, nil
	case string(models.ResourceOrgUser):
		return models.ResourceOrgUser, value, nil
	default:
		return models.Resource(""), "", fmt.Errorf("unsupported resource: %s", resource)
	}
}

func handleListOrgMemberTypesV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(userAuthRequestContext).(userIdentity)
	log(common.LogLevelDebug, fmt.Sprintf("user[%s] requested member types for orgs", session.UserId))
	memberTypes := []string{}
	for memberType := range models.OrgMemberTypeMap {
		memberTypes = append(memberTypes, memberType)
	}

	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.List,
		ResourceType: audit.OrgMemberTypesResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", memberTypes)
}

type handleOrgTokenValidationV1Input struct {
	TokenId string `json:"tokenId"`
	Token   string `json:"token"`
}

// TODO
func handleOrgTokenValidationV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelInfo, "hi")
	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", types.ErrorInvalidInput)
		return
	}
	var input handleOrgTokenValidationV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", types.ErrorInvalidInput)
		return
	}

	_, err = models.ValidateOrgTokenV1(models.ValidateOrgTokenV1Opts{
		DatabaseConnection: models.DatabaseConnection{Db: dbInstance},
		Token:              input.Token,
		TokenId:            input.TokenId,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "")
}
