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
	"opsicle/internal/controller/models"
	"opsicle/internal/controller/templates"
	"opsicle/internal/email"
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
	v1.Handle("/invitation/{invitationId}", requiresAuth(http.HandlerFunc(handleUpdateOrgInvitationV1))).Methods(http.MethodPatch)

	v1 = opts.Router.PathPrefix("/v1/orgs").Subrouter()

	v1.Handle("", requiresAuth(http.HandlerFunc(handleListOrgsV1))).Methods(http.MethodGet)
}

type handleCreateOrgV1Output struct {
	Id   string `json:"id"`
	Code string `json:"code"`
}

type handleCreateOrgV1Input struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// handleCreateOrgV1 godoc
// @Summary      Creates a new organisation
// @Description  Creates a new organisation and assigns the user identified by their token as the administrator of the organisation
// @Tags         controller-service
// @Accept       json
// @Produce      json
// @Param        request body handleCreateOrgV1Input true "User credentials"
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
	var input handleCreateOrgV1Input
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

	orgId, err := models.CreateOrgV1(models.CreateOrgV1Opts{
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
	log(common.LogLevelDebug, fmt.Sprintf("successfully created org[%s] with id[%s]", input.Code, orgId))
	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.Create,
		ResourceId:   orgId,
		ResourceType: audit.OrgResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleCreateOrgV1Output{
		Id:   orgId,
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
	log(common.LogLevelDebug, fmt.Sprintf("successfully retrieved user[%s] in org[%s]", orgUser.UserId, orgUser.OrgId))
	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.Get,
		ResourceId:   orgUser.UserId,
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
	JoinedAt   time.Time `json:"joinedAt"`
	MemberType string    `json:"memberType"`
	OrgId      string    `json:"orgId"`
	OrgCode    string    `json:"orgCode"`
	OrgName    string    `json:"orgName"`
	UserId     string    `json:"userId"`
	UserEmail  string    `json:"userEmail"`
	UserType   string    `json:"userType"`
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
		output = append(
			output,
			handleListOrgUsersV1OutputUser{
				JoinedAt:   orgUser.JoinedAt,
				MemberType: orgUser.MemberType,
				OrgId:      orgUser.OrgId,
				OrgCode:    orgUser.OrgCode,
				OrgName:    orgUser.OrgName,
				UserId:     orgUser.UserId,
				UserEmail:  orgUser.UserEmail,
				UserType:   orgUser.UserType,
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
		OrgCode:    orgUser.OrgCode,
		OrgId:      orgUser.OrgId,
		UserId:     orgUser.UserId,

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

		if err := orgInvite.DeleteById(models.DatabaseConnection{Db: db}); err != nil {
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
			OrgId:          orgUser.OrgId,
			OrgCode:        orgUser.OrgCode,
			OrgName:        orgUser.OrgName,
			UserId:         orgUser.UserId,
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
		if err := orgInvite.DeleteById(models.DatabaseConnection{Db: db}); err != nil {
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

	orgUser := models.OrgUser{
		OrgId:  orgId,
		UserId: session.UserId,
	}
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

	orgUser := models.OrgUser{
		OrgId:  orgId,
		UserId: userId,
	}
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

	orgUser := models.OrgUser{OrgId: orgId, UserId: userId}
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
