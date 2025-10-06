package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/audit"
	"opsicle/internal/common"
	"opsicle/internal/common/images"
	"opsicle/internal/controller/constants"
	"opsicle/internal/controller/models"
	"opsicle/internal/controller/templates"
	"opsicle/internal/email"
	"opsicle/internal/tls"
	"opsicle/internal/validate"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func registerOrgRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1/org").Subrouter()

	v1.Handle("/member/types", requiresAuth(http.HandlerFunc(handleListOrgMemberTypesV1))).Methods(http.MethodGet)
	v1.Handle("", requiresAuth(http.HandlerFunc(handleCreateOrgV1))).Methods(http.MethodPost)
	v1.Handle("/{orgCode}", requiresAuth(http.HandlerFunc(handleGetOrgV1))).Methods(http.MethodGet)
	v1.Handle("/{orgId}/member", requiresAuth(http.HandlerFunc(handleCreateOrgUserV1))).Methods(http.MethodPost)
	v1.Handle("/{orgId}/member", requiresAuth(http.HandlerFunc(handleGetOrgCurrentUserV1))).Methods(http.MethodGet)
	v1.Handle("/{orgId}/member", requiresAuth(http.HandlerFunc(handleUpdateOrgUserV1))).Methods(http.MethodPatch)
	v1.Handle("/{orgId}/member", requiresAuth(http.HandlerFunc(handleLeaveOrgV1))).Methods(http.MethodDelete)
	v1.Handle("/{orgId}/member/{userId}", requiresAuth(http.HandlerFunc(handleDeleteOrgUserV1))).Methods(http.MethodDelete)
	v1.Handle("/{orgId}/members", requiresAuth(http.HandlerFunc(handleListOrgUsersV1))).Methods(http.MethodGet)
	v1.Handle("/{orgId}/roles", requiresAuth(http.HandlerFunc(handleListOrgRolesV1))).Methods(http.MethodGet)
	v1.Handle("/{orgId}/tokens", requiresAuth(http.HandlerFunc(handleListOrgTokensV1))).Methods(http.MethodGet)
	v1.Handle("/{orgId}/token/{tokenId}", requiresAuth(http.HandlerFunc(handleGetOrgTokenV1))).Methods(http.MethodGet)
	v1.Handle("/{orgId}/token", requiresAuth(http.HandlerFunc(handleCreateOrgTokenV1))).Methods(http.MethodPost)
	v1.Handle("/invitation/{invitationId}", requiresAuth(http.HandlerFunc(handleUpdateOrgInvitationV1))).Methods(http.MethodPatch)

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
	session := r.Context().Value(authRequestContext).(identity)
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, "successfully read body into bytes")
	var input CreateOrgV1Input
	if err := json.Unmarshal(requestBody, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, "successfully parsed body into expected input class")

	if err := validate.OrgName(input.Name); err != nil {
		log(common.LogLevelDebug, fmt.Sprintf("user[%s] entered an invalid orgName[%s]: %s", session.UserId, input.Name, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "org name is invalid", ErrorInvalidInput, err.Error())
		return
	}
	if err := validate.OrgCode(input.Code); err != nil {
		log(common.LogLevelDebug, fmt.Sprintf("user[%s] entered an invalid orgCode[%s]: %s", session.UserId, input.Code, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "org code is invalid", ErrorInvalidInput, err.Error())
		return
	}

	orgInstance, err := models.CreateOrgV1(models.CreateOrgV1Opts{
		Db:     db,
		Code:   input.Code,
		Name:   input.Name,
		Type:   models.TypeTenantOrg,
		UserId: session.UserId,
	})
	if err != nil {
		if errors.Is(err, models.ErrorDuplicateEntry) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "org already exists", ErrorOrgExists)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create org", ErrorDatabaseIssue)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("created org[%s] with id[%s]", input.Code, orgInstance.GetId()))
	log(common.LogLevelDebug, fmt.Sprintf("adding user[%s] to org[%s]", session.UserId, orgInstance.GetId()))
	if err := orgInstance.AddUserV1(models.AddUserToOrgV1{
		Db:         db,
		UserId:     session.UserId,
		MemberType: string(models.TypeOrgAdmin),
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to add user[%s] to org[%s]: %s", session.UserId, orgInstance.GetId(), err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to add user to org", ErrorDatabaseIssue)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("added user[%s] to org[%s] as admin", session.UserId, orgInstance.GetId()))

	log(common.LogLevelDebug, fmt.Sprintf("adding default role to org[%s]", orgInstance.GetId()))
	orgRole, err := orgInstance.CreateRoleV1(models.CreateOrgRoleV1Input{
		DatabaseConnection: models.DatabaseConnection{Db: db},
		RoleName:           models.DefaultOrgRoleName,
		UserId:             session.UserId,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to add default role to org[%s]: %s", orgRole.GetId(), err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to add role to org", ErrorDatabaseIssue)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("added default role[%s] to org[%s]", orgRole.GetId(), orgInstance.GetId()))

	log(common.LogLevelDebug, fmt.Sprintf("adding default permissions to orgRole[%s]", orgRole.GetId()))
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
		if err := orgRole.CreatePermissionV1(models.CreateOrgRolePermissionV1Input{
			Allows:             models.ActionSetAdmin,
			Resource:           resource,
			DatabaseConnection: models.DatabaseConnection{Db: db},
		}); err != nil {
			permissionErrs = append(permissionErrs, err)
		}
	}
	if len(permissionErrs) > 0 {
		log(common.LogLevelError, fmt.Sprintf("failed to add permissions to orgRole[%s]: %s", orgRole.GetId(), errors.Join(permissionErrs...)))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to add permissions to org role", ErrorDatabaseIssue)
		return
	}
	assigner := session.UserId
	if err := orgRole.AssignUserV1(models.AssignOrgRoleV1Input{
		DatabaseConnection: models.DatabaseConnection{Db: db},
		OrgId:              orgInstance.GetId(),
		UserId:             session.UserId,
		AssignedBy:         &assigner,
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to assign default role[%s] to user[%s]: %s", orgRole.GetId(), session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to assign org role", ErrorDatabaseIssue)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("creating certificate authority for org[%s]", orgInstance.GetId()))
	if _, err := orgInstance.CreateCertificateAuthorityV1(models.CreateOrgCertificateAuthorityV1Input{
		DatabaseConnection: models.DatabaseConnection{Db: db},
		CertOptions: &tls.CertificateOptions{
			NotAfter: time.Now().Add(time.Hour * 24 * 365 * 5),
		},
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to create certificate authority for org[%s]: %s", orgInstance.GetId(), err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create certificate authority", ErrorDatabaseIssue)
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
	session := r.Context().Value(authRequestContext).(identity)

	vars := mux.Vars(r)
	orgCode := vars["orgCode"]

	if err := validate.OrgCode(orgCode); err != nil {
		log(common.LogLevelDebug, fmt.Sprintf("user[%s] entered an invalid orgCode[%s]: %s", session.UserId, orgCode, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "org code is invalid", ErrorInvalidInput)
		return
	}

	// retrieve the org

	org, err := models.GetOrgV1(models.GetOrgV1Opts{
		Db:   db,
		Code: &orgCode,
	})
	if err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org", ErrorDatabaseIssue)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org", ErrorDatabaseIssue)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("successfully retrieved org[%s] with id[%s]", orgCode, org.GetId()))

	// only return the information if the user is part of the org

	orgUser, err := org.GetUserV1(models.GetOrgUserV1Opts{
		Db:     db,
		UserId: session.UserId,
	})
	if err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			log(common.LogLevelError, fmt.Sprintf("unauthorized user[%s] requested data about org[%s]: %s", session.UserId, org.GetId(), err))
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org", ErrorDatabaseIssue)
			return
		}
		log(common.LogLevelError, fmt.Sprintf("failed to retrieve user[%s] in org[%s]: %s", session.UserId, org.GetId(), err))
		common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org", ErrorDatabaseIssue)
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
	session := r.Context().Value(authRequestContext).(identity)

	log(common.LogLevelDebug, fmt.Sprintf("retrieving organisations that user[%s] is in", session.UserId))

	orgs, err := models.ListUserOrgsV1(models.ListUserOrgsV1Opts{
		Db:     db,
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
	session := r.Context().Value(authRequestContext).(identity)
	vars := mux.Vars(r)
	orgId := vars["orgId"]

	log(common.LogLevelDebug, fmt.Sprintf("user[%s] requested list of users from org[%s]", session.UserId, orgId))
	org := models.Org{Id: &orgId}
	_, err := org.GetUserV1(models.GetOrgUserV1Opts{Db: db, UserId: session.UserId})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to get user[%s] from org[%s]: %s", session.UserId, orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, fmt.Sprintf("refused to list users in org[%s] at user[%s]'s request", orgId, session.UserId), ErrorInvalidInput)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org users", ErrorDatabaseIssue)
		return
	}
	orgUsers, err := org.ListUsersV1(models.DatabaseConnection{Db: db})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to list users from org[%s]: %s", orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org users", ErrorDatabaseIssue)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org users", ErrorDatabaseIssue)
		return
	}
	output := handleListOrgUsersV1Output{}
	for _, orgUser := range orgUsers {
		userRoles, err := orgUser.ListRolesV1(models.DatabaseConnection{Db: db})
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to list roles for user[%s] in org[%s]: %s", orgUser.User.GetId(), orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org user roles", ErrorDatabaseIssue)
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
	session := r.Context().Value(authRequestContext).(identity)
	vars := mux.Vars(r)
	orgId := vars["orgId"]

	log(common.LogLevelDebug, fmt.Sprintf("user[%s] requested list of roles from org[%s]", session.UserId, orgId))
	org := models.Org{Id: &orgId}
	if _, err := org.GetUserV1(models.GetOrgUserV1Opts{Db: db, UserId: session.UserId}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to get user[%s] from org[%s]: %s", session.UserId, orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, fmt.Sprintf("refused to list roles in org[%s] at user[%s]'s request", orgId, session.UserId), ErrorInvalidInput)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org roles", ErrorDatabaseIssue)
		return
	}
	orgRoles, err := org.ListRolesV1(models.DatabaseConnection{Db: db})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to list roles from org[%s]: %s", orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org roles", ErrorDatabaseIssue)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org roles", ErrorDatabaseIssue)
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
	session := r.Context().Value(authRequestContext).(identity)
	vars := mux.Vars(r)
	orgId := vars["orgId"]

	log(common.LogLevelDebug, fmt.Sprintf("user[%s] requested list of tokens from org[%s]", session.UserId, orgId))

	if err := validateRequesterCanManageOrgUsers(validateRequesterCanManageOrgUsersOpts{
		OrgId:           orgId,
		RequesterUserId: session.UserId,
	}); err != nil {
		switch {
		case errors.Is(err, ErrorInsufficientPermissions):
			log(common.LogLevelError, fmt.Sprintf("user[%s] is not authorized to list tokens for org[%s]", session.UserId, orgId))
			common.SendHttpFailResponse(w, r, http.StatusForbidden, "requester is not authorized to list tokens", ErrorInsufficientPermissions)
			return
		case errors.Is(err, ErrorDatabaseIssue):
			log(common.LogLevelError, fmt.Sprintf("failed to verify requester permissions for org[%s]: %s", orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester permissions", ErrorDatabaseIssue)
			return
		default:
			log(common.LogLevelError, fmt.Sprintf("unexpected error verifying requester permissions for org[%s]: %s", orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester permissions", ErrorUnknown)
			return
		}
	}

	org := models.Org{Id: &orgId}
	orgTokens, err := org.ListTokensV1(models.ListOrgTokensV1Opts{DatabaseConnection: models.DatabaseConnection{Db: db}})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to list tokens for org[%s]: %s", orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org tokens", ErrorDatabaseIssue)
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
	session := r.Context().Value(authRequestContext).(identity)
	vars := mux.Vars(r)
	orgId := vars["orgId"]
	tokenId := vars["tokenId"]

	if err := validate.Uuid(tokenId); err != nil {
		log(common.LogLevelDebug, fmt.Sprintf("user[%s] provided invalid tokenId[%s]: %s", session.UserId, tokenId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "token id is invalid", ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("user[%s] requested token[%s] from org[%s]", session.UserId, tokenId, orgId))

	if err := validateRequesterCanManageOrgUsers(validateRequesterCanManageOrgUsersOpts{
		OrgId:           orgId,
		RequesterUserId: session.UserId,
	}); err != nil {
		switch {
		case errors.Is(err, ErrorInsufficientPermissions):
			log(common.LogLevelError, fmt.Sprintf("user[%s] is not authorized to view token[%s] for org[%s]", session.UserId, tokenId, orgId))
			common.SendHttpFailResponse(w, r, http.StatusForbidden, "requester is not authorized to view token", ErrorInsufficientPermissions)
			return
		case errors.Is(err, ErrorDatabaseIssue):
			log(common.LogLevelError, fmt.Sprintf("failed to verify requester permissions for org[%s]: %s", orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester permissions", ErrorDatabaseIssue)
			return
		default:
			log(common.LogLevelError, fmt.Sprintf("unexpected error verifying requester permissions for org[%s]: %s", orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester permissions", ErrorUnknown)
			return
		}
	}

	org := models.Org{Id: &orgId}
	orgToken, err := org.GetTokenByIdV1(models.GetOrgTokenByIdV1Opts{
		DatabaseConnection: models.DatabaseConnection{Db: db},
		TokenId:            tokenId,
	})
	if err != nil {
		switch {
		case errors.Is(err, models.ErrorNotFound):
			log(common.LogLevelDebug, fmt.Sprintf("token[%s] in org[%s] not found", tokenId, orgId))
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "token was not found", ErrorNotFound)
			return
		default:
			log(common.LogLevelError, fmt.Sprintf("failed to load token[%s] for org[%s]: %s", tokenId, orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org token", ErrorDatabaseIssue)
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

type handleCreateOrgTokenV1Input struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	RoleId      string  `json:"roleId"`
}

type handleCreateOrgTokenV1Output struct {
	TokenId     string `json:"tokenId"`
	Name        string `json:"name"`
	ApiKey      string `json:"apiKey"`
	Certificate string `json:"certificatePem"`
	PrivateKey  string `json:"privateKeyPem"`
}

func handleCreateOrgTokenV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
	vars := mux.Vars(r)
	orgId := vars["orgId"]

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", ErrorInvalidInput)
		return
	}
	var input handleCreateOrgTokenV1Input
	if err := json.Unmarshal(requestBody, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", ErrorInvalidInput)
		return
	}
	if input.Name == "" {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "token name is required", ErrorInvalidInput)
		return
	}
	if input.RoleId == "" {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "role id is required", ErrorInvalidInput)
		return
	}
	if err := validateRequesterCanManageOrgUsers(validateRequesterCanManageOrgUsersOpts{
		OrgId:           orgId,
		RequesterUserId: session.UserId,
	}); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "requester is not authorized to manage tokens", ErrorInsufficientPermissions)
		return
	}

	org := models.Org{Id: &orgId}
	orgDetails, err := models.GetOrgV1(models.GetOrgV1Opts{Db: db, Id: &orgId})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to get org[%s]: %s", orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org", ErrorNotFound)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org", ErrorDatabaseIssue)
		return
	}

	orgRole, err := org.GetRoleByIdV1(models.GetOrgRoleByIdV1Opts{
		DatabaseConnection: models.DatabaseConnection{Db: db},
		RoleId:             input.RoleId,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to load role[%s] for org[%s]: %s", input.RoleId, orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusNotFound, "specified role was not found", ErrorNotFound)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to load org role", ErrorDatabaseIssue)
		return
	}

	ca, err := org.LoadCertificateAuthorityV1(models.LoadOrgCertificateAuthorityV1Opts{
		DatabaseConnection: models.DatabaseConnection{Db: db},
	})
	if err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			log(common.LogLevelDebug, fmt.Sprintf("no certificate authority for org[%s], creating new one", orgId))
			ca, err = org.CreateCertificateAuthorityV1(models.CreateOrgCertificateAuthorityV1Input{
				DatabaseConnection: models.DatabaseConnection{Db: db},
			})
		}
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to prepare certificate authority for org[%s]: %s", orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to prepare certificate authority", ErrorDatabaseIssue)
			return
		}
	}

	caCert, caKey, err := ca.GetCryptoMaterials()
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to parse certificate authority for org[%s]: %s", orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to load certificate authority", ErrorDatabaseIssue)
		return
	}

	apiKey, err := generateApiKey(constants.ApiKeyLength - len(constants.ApiKeyPrefix))
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to generate api key for org[%s]: %s", orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to generate api key", ErrorUnknown)
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
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to generate certificate", ErrorUnknown)
		return
	}

	createdBy := session.UserId
	orgToken, err := org.CreateTokenV1(models.CreateOrgTokenV1Input{
		TokenId:            tokenId,
		DatabaseConnection: models.DatabaseConnection{Db: db},
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
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create org token", ErrorDatabaseIssue)
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

	output := handleCreateOrgTokenV1Output{
		TokenId:     orgToken.GetId(),
		Name:        orgToken.Name,
		ApiKey:      apiKey,
		Certificate: string(certificate.Pem),
		PrivateKey:  string(key.Pem),
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
	session := r.Context().Value(authRequestContext).(identity)

	vars := mux.Vars(r)
	orgId := vars["orgId"]

	log(common.LogLevelDebug, fmt.Sprintf("user[%s] is adding another user to org[%s]", session.UserId, orgId))

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, "successfully read body into bytes")
	var input handleCreateOrgUserV1Input
	if err := json.Unmarshal(requestBody, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, "successfully parsed body into expected input class")

	if err := validate.Email(input.Email); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to validate invitee's email: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to receive valid email", ErrorInvalidInput)
		return
	}

	org, err := models.GetOrgV1(models.GetOrgV1Opts{
		Db: db,
		Id: &orgId,
	})
	if err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to validate org", ErrorInvalidInput)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org", ErrorDatabaseIssue)
		return
	}

	if err := validateRequesterCanManageOrgUsers(validateRequesterCanManageOrgUsersOpts{
		OrgId:           orgId,
		RequesterUserId: session.UserId,
	}); err != nil {
		switch true {
		case errors.Is(err, ErrorInsufficientPermissions):
			log(common.LogLevelError, fmt.Sprintf("user[%s] doesn't have permissions to add a member to org[%s]: %s", session.UserId, orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to verify requester", ErrorInsufficientPermissions)
			return
		case errors.Is(err, ErrorDatabaseIssue):
			log(common.LogLevelError, fmt.Sprintf("encountered database issue while processing request by user[%s] to add user with email[%s] to org[%s]: %s", session.UserId, input.Email, orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester", ErrorDatabaseIssue)
			return
		}
	}

	isAcceptorExists := false
	acceptor := models.User{Email: input.Email}
	if err := acceptor.LoadByEmailV1(models.DatabaseConnection{Db: db}); err != nil {
		if !errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve acceptor", ErrorDatabaseIssue)
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
		Db:             db,
		InviterId:      session.UserId,
		JoinCode:       joinCode,
		MembershipType: input.Type,
	}
	if isAcceptorExists {
		invitationOpts.AcceptorId = acceptor.Id
		if _, err = org.GetUserV1(models.GetOrgUserV1Opts{
			Db:     db,
			UserId: session.UserId,
		}); err != nil {
			if !errors.Is(err, models.ErrorNotFound) {
				common.SendHttpFailResponse(w, r, http.StatusBadRequest, "user already in org", ErrorUserExistsInOrg)
				return
			}
		}
	} else {
		invitationOpts.AcceptorEmail = &input.Email
	}

	invitationOutput, err := org.InviteUserV1(invitationOpts)
	if err != nil {
		if errors.Is(err, models.ErrorDuplicateEntry) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invitation exists", ErrorInvitationExists)
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
						publicServerUrl,
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
	session := r.Context().Value(authRequestContext).(identity)

	vars := mux.Vars(r)
	orgId := vars["orgId"]
	if _, err := uuid.Parse(orgId); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid org id: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid org id", ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("received request by user[%s] to get information on their membership in org[%s]", session.UserId, orgId))

	org := models.Org{Id: &orgId}
	orgUser, err := org.GetUserV1(models.GetOrgUserV1Opts{
		Db:     db,
		UserId: session.UserId,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to retrieve user[%s] from org[%s]: %s", session.UserId, orgId, err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get user", ErrorInvalidInput)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to get user", ErrorDatabaseIssue)
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
	session := r.Context().Value(authRequestContext).(identity)
	vars := mux.Vars(r)
	invitationId := vars["invitationId"]

	log(common.LogLevelDebug, fmt.Sprintf("updating user[%s]'s organisation invite[%s]", session.UserId, invitationId))

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", ErrorInvalidInput)
		return
	}

	var input handleUpdateOrgInvitationV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", ErrorInvalidInput)
		return
	}

	orgInvite := models.OrgUserInvitation{Id: invitationId}
	if err := orgInvite.LoadV1(models.DatabaseConnection{Db: db}); err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed find invitation", ErrorInvitationInvalid)
			return
		}
		log(common.LogLevelError, fmt.Sprintf("failed to load invitation: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to load", ErrorInvalidInput)
		return
	}

	if !strings.EqualFold(*orgInvite.AcceptorId, session.UserId) {
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "user not allowed", ErrorInvitationInvalid)
		return
	}
	if strings.Compare(orgInvite.JoinCode, input.JoinCode) != 0 {
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "invalid join code", ErrorInvitationInvalid)
		return
	}

	if input.IsAcceptance {
		org := models.Org{Id: &orgInvite.OrgId}
		if err := org.AddUserV1(models.AddUserToOrgV1{
			Db:         db,
			UserId:     session.UserId,
			MemberType: orgInvite.Type,
		}); err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to add user to org: %s", err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to add user", ErrorDatabaseIssue)
			return
		}

		if err := orgInvite.DeleteByIdV1(models.DatabaseConnection{Db: db}); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to delete invitation", ErrorDatabaseIssue)
			return
		}

		orgUser, err := org.GetUserV1(models.GetOrgUserV1Opts{
			Db:     db,
			UserId: session.UserId,
		})
		if err != nil {
			if errors.Is(err, models.ErrorNotFound) {
				common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to retrieve org user", ErrorDatabaseIssue)
				return
			}
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org user", ErrorDatabaseIssue)
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
		if err := orgInvite.DeleteByIdV1(models.DatabaseConnection{Db: db}); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to delete invitation", ErrorDatabaseIssue)
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
	session := r.Context().Value(authRequestContext).(identity)

	vars := mux.Vars(r)
	orgId := vars["orgId"]
	if _, err := uuid.Parse(orgId); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid org id: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid org id", ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("received request by user[%s] to leave org[%s]", session.UserId, orgId))

	orgUser := models.NewOrgUser()
	orgUser.Org.Id = &orgId
	orgUser.User.Id = &session.UserId
	if err := orgUser.LoadV1(models.DatabaseConnection{Db: db}); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to load user to be removed", ErrorDatabaseIssue)
		return
	}
	if err := validateUserIsNotLastAdmin(validateUserIsNotLastAdminOpts{
		OrgId:  orgId,
		UserId: session.UserId,
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] is the last admin and cannot leave the organisation: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "user is last admin", ErrorOrgRequiresOneAdmin)
		return
	}

	// delete the org user

	if err := orgUser.DeleteV1(models.DatabaseConnection{Db: db}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to remove user[%s] from org[%s]: %s", session.UserId, orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to remove member", ErrorDatabaseIssue)
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
	session := r.Context().Value(authRequestContext).(identity)

	vars := mux.Vars(r)
	orgId := vars["orgId"]
	if _, err := uuid.Parse(orgId); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid org id: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid org id", ErrorInvalidInput)
		return
	}

	userId := vars["userId"]
	if _, err := uuid.Parse(userId); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid user id: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid user id", ErrorInvalidInput)
		return
	}
	if userId == session.UserId {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "a user cannot delete itself", ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("received request by user[%s] to delete user[%s] from org[%s]", session.UserId, userId, orgId))

	// verify requester can update an org user

	if err := validateRequesterCanManageOrgUsers(validateRequesterCanManageOrgUsersOpts{
		OrgId:           orgId,
		RequesterUserId: session.UserId,
	}); err != nil {
		switch true {
		case errors.Is(err, ErrorInsufficientPermissions):
			log(common.LogLevelError, fmt.Sprintf("user[%s] doesn't have permissions to delete user[%s] from org[%s]: %s", session.UserId, userId, orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to verify requester", ErrorInsufficientPermissions)
			return
		case errors.Is(err, ErrorDatabaseIssue):
			log(common.LogLevelError, fmt.Sprintf("encountered database issue while processing request by user[%s] to delete user[%s] from org[%s]: %s", session.UserId, userId, orgId, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester", ErrorDatabaseIssue)
			return
		}
	}
	log(common.LogLevelDebug, fmt.Sprintf("validated user[%s] is able to manage org[%s] members", session.UserId, orgId))

	orgUser := models.NewOrgUser()
	orgUser.Org.Id = &orgId
	orgUser.User.Id = &session.UserId
	if err := orgUser.LoadV1(models.DatabaseConnection{Db: db}); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to load user to be removed", ErrorDatabaseIssue)
		return
	}
	if err := validateUserIsNotLastAdmin(validateUserIsNotLastAdminOpts{
		OrgId:  orgId,
		UserId: session.UserId,
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] is the last admin and cannot leave the organisation: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "user is last admin", ErrorOrgRequiresOneAdmin)
		return
	}

	// delete the org user

	if err := orgUser.DeleteV1(models.DatabaseConnection{Db: db}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to remove user[%s] from org[%s]: %s", userId, orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to remove member", ErrorDatabaseIssue)
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

type handleUpdateOrgUserV1Output struct {
	IsSuccessful bool `json:"isSuccessful"`
}

type handleUpdateOrgUserV1Input struct {
	Update map[string]any `json:"update"`
	User   string         `json:"user"`
}

func handleUpdateOrgUserV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)

	vars := mux.Vars(r)
	orgId := vars["orgId"]
	if _, err := uuid.Parse(orgId); err != nil {
		log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid org id: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid org id", ErrorInvalidInput)
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", ErrorInvalidInput)
		return
	}

	var input handleUpdateOrgUserV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", ErrorInvalidInput)
		return
	}

	isUserEmail := false
	if _, err := uuid.Parse(input.User); err != nil {
		if err := validate.Email(input.User); err != nil {
			log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid user id: %s", session.UserId, err))
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid user id", ErrorInvalidInput)
			return
		} else {
			isUserEmail = true
		}
	}

	userId := ""

	if isUserEmail {
		user := models.User{Email: input.User}
		if err := user.LoadByEmailV1(models.DatabaseConnection{Db: db}); err != nil {
			log(common.LogLevelError, fmt.Sprintf("user[%s] submitted an invalid user email: %s", session.UserId, err))
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid user identifier", ErrorInvalidInput)
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
		case errors.Is(err, ErrorInsufficientPermissions):
			common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to verify requester", ErrorInsufficientPermissions)
			return
		case errors.Is(err, ErrorDatabaseIssue):
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester", ErrorDatabaseIssue)
			return
		}
	}

	orgUser := models.NewOrgUser()
	orgUser.Org.Id = &orgId
	orgUser.User.Id = &session.UserId
	if err := orgUser.LoadV1(models.DatabaseConnection{Db: db}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to load user[%s] in org[%s]: %s", userId, orgId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve org user", ErrorDatabaseIssue)
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
					Db:   db,
					Role: models.TypeOrgAdmin,
				})
				if err != nil {
					log(common.LogLevelError, fmt.Sprintf("encountered database issue while retrieving member count from org[%s]: %s", orgId, err))
					common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify requester", ErrorDatabaseIssue)
					return
				}
				if adminsCount == 1 {
					common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to update membership type of the only administrator", ErrorInvalidInput)
					return
				}
			}
		}
	}

	// make org user changes

	if err := orgUser.UpdateFieldsV1(models.UpdateFieldsV1{
		Db:          db,
		FieldsToSet: input.Update,
	}); err != nil {
		if errors.Is(err, models.ErrorInvalidInput) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to update org user", ErrorInvalidInput)
			return
		}
		log(common.LogLevelError, fmt.Sprintf("request by user[%s] to update user[%s] in org[%s] failed: %s", session.UserId, userId, orgId, err))
		if errors.Is(err, models.ErrorInvalidInput) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to update org user", ErrorInvalidInput)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to update org user", ErrorDatabaseIssue)
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

func handleListOrgMemberTypesV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
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
