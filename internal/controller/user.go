package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/audit"
	"opsicle/internal/auth"
	"opsicle/internal/common"
	"opsicle/internal/common/images"
	"opsicle/internal/controller/models"
	"opsicle/internal/controller/templates"
	"opsicle/internal/email"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func registerUserRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1/users").Subrouter()

	v1.HandleFunc("", handleCreateUserV1).Methods(http.MethodPost)

	v1 = opts.Router.PathPrefix("/v1/user").Subrouter()

	v1.Handle("/logs", requiresAuth(http.HandlerFunc(handleListUserAuditLogsV1))).Methods(http.MethodGet)
	v1.Handle("/mfa", requiresAuth(http.HandlerFunc(handleCreateUserMfaV1))).Methods(http.MethodPost)
	v1.Handle("/mfa/{mfaId}", requiresAuth(http.HandlerFunc(handleVerifyUserMfaV1))).Methods(http.MethodPost)
	v1.Handle("/mfas", requiresAuth(http.HandlerFunc(handleListUserMfasV1))).Methods(http.MethodGet)
	v1.HandleFunc("/mfas", handleListUserMfaTypesV1).Methods(http.MethodOptions)
	v1.Handle("/org-invitations", requiresAuth(http.HandlerFunc(handleListUserOrgInvitationsV1))).Methods(http.MethodGet)
	v1.HandleFunc("/password", handleUpdateUserPasswordV1).Methods(http.MethodPatch)

	v1 = opts.Router.PathPrefix("/v1/verification").Subrouter()

	v1.HandleFunc("/{verificationCode}", handleVerifyUserV1).Methods(http.MethodGet)
}

type handleListUserAuditLogsV1Input struct {
	Cursor  time.Time `json:"cursor"`
	Limit   int64     `json:"limit"`
	Reverse bool      `json:"reverse"`
}

type handleListUserAuditLogsV1Output models.AuditLogs

func handleListUserAuditLogsV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", ErrorInvalidInput)
		return
	}
	var input handleListUserAuditLogsV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("retrieving audit logs for user[%s]", session.UserId))

	user := models.User{Id: &session.UserId}
	auditLogs, err := user.ListAuditLogsV1(models.ListAuditLogsV1Opts{
		Timestamp: input.Cursor,
		Limit:     input.Limit,
		Reverse:   input.Reverse,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve audit logs", ErrorDatabaseIssue)
		return
	}

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleListUserAuditLogsV1Output(auditLogs))
}

type handleCreateUserV1Input struct {
	// OrgInviteCode if present, allows the automatic registering of the user
	// with the organisation. Each invite code is only valid once
	OrgInviteCode *string `json:"orgInviteCode"`

	// Email is the user's email address
	Email string `json:"email"`

	// Password is the user's password
	Password string `json:"password"`
}

type handleCreateUserV1Output struct {
	Id    string `json:"id"`
	Email string `json:"email"`
}

func handleCreateUserV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelDebug, "this creates a new user")

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to read request body: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", err)
		return
	}
	var input handleCreateUserV1Input
	if err := json.Unmarshal(requestBody, &input); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to unmarshal request body: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", err)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("processing request to create user[%s]", input.Email))

	if _, err := auth.IsEmailValid(input.Email); err != nil {
		log(common.LogLevelError, "invalid email entered")
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid email address", err)
		return
	}
	if _, err := auth.IsPasswordValid(input.Password); err != nil {
		log(common.LogLevelError, "invalid password entered")
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid password", err)
		return
	}

	if err := models.CreateUserV1(models.CreateUserV1Opts{
		Db: db,

		Email:    input.Email,
		Password: input.Password,
		Type:     models.TypeUser,
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to create user: %s", err))
		if errors.Is(err, models.ErrorDuplicateEntry) {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create user, email already exists", ErrorEmailExists)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create user for unexpected reasons", err)
		return
	}

	user := models.User{Email: input.Email}
	if err := user.LoadByEmailV1(models.DatabaseConnection{Db: db}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to retrieve user via email: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve user", err)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("listing organisation invitations to user's email address[%s]...", input.Email))
	listOrgInvitations, err := models.ListOrgInvitationsV1(models.ListOrgInvitationsV1Opts{
		Db:        db,
		UserEmail: &input.Email,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to list user's org invitations: %s", err))
		if !errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve user", err)
			return
		}
	}
	for _, orgInvitation := range listOrgInvitations {
		var orgInvitationReplacementErrs []error
		orgInvitation.AcceptorId = user.Id
		if err := orgInvitation.ReplaceAcceptorEmailWithId(models.DatabaseConnection{Db: db}); err != nil {
			orgInvitationReplacementErrs = append(orgInvitationReplacementErrs, err)
		}
		if len(orgInvitationReplacementErrs) > 0 {
			log(common.LogLevelError, fmt.Sprintf("failed to process org invitations for user[%s]: %s", *user.Id, errors.Join(orgInvitationReplacementErrs...)))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to convert user's org invitations", err)
			return
		}
	}

	log(common.LogLevelDebug, fmt.Sprintf("listing template invitations to user's email address[%s]...", input.Email))
	listTemplateInvitations, err := models.ListTemplateInvitationsV1(models.ListTemplateInvitationsV1Opts{
		Db:        db,
		UserEmail: &input.Email,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to list user's template invitations: %s", err))
		if !errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve user", err)
			return
		}
	}
	for _, templateInvitation := range listTemplateInvitations {
		var templateInvitationReplacementErrs []error
		templateInvitation.AcceptorId = user.Id
		if err := templateInvitation.ReplaceAcceptorEmailWithId(models.DatabaseConnection{Db: db}); err != nil {
			templateInvitationReplacementErrs = append(templateInvitationReplacementErrs, err)
		}
		if len(templateInvitationReplacementErrs) > 0 {
			log(common.LogLevelError, fmt.Sprintf("failed to process template invitations for user[%s]: %s", *user.Id, errors.Join(templateInvitationReplacementErrs...)))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to convert user's template invitations", err)
			return
		}
	}

	if !user.IsVerified() {
		if smtpConfig.IsSet() {
			remoteAddr := r.RemoteAddr
			userAgent := r.UserAgent()
			opsicleCatMimeType, opsicleCatData := images.GetOpsicleCat()
			if err := email.SendSmtp(email.SendSmtpOpts{
				ServiceLogs: *serviceLogs,
				To: []email.User{
					{
						Address: user.Email,
					},
				},
				Sender: smtpConfig.Sender,
				Message: email.Message{
					Title: "Verify your Opsicle account email to get started",
					Body: templates.GetEmailVerificationMessage(
						publicServerUrl,
						user.EmailVerificationCode,
						remoteAddr,
						userAgent,
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
				log(common.LogLevelWarn, fmt.Sprintf("failed to send email, send user their verification code[%s] manually", user.EmailVerificationCode))
			}
		} else {
			log(common.LogLevelWarn, fmt.Sprintf("smtp is not available, send user their verification code[%s] manually", user.EmailVerificationCode))
		}
	}

	userAgent := r.UserAgent()
	audit.Log(audit.LogEntry{
		EntityId:     *user.Id,
		EntityType:   audit.UserEntity,
		Verb:         audit.Create,
		ResourceId:   *user.Id,
		ResourceType: audit.UserResource,
		Status:       audit.Success,
		SrcIp:        &r.RemoteAddr,
		SrcUa:        &userAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleCreateUserV1Output{
		Id:    *user.Id,
		Email: user.Email,
	})
}

type handleCreateUserMfaV1Input struct {
	Password string `json:"password"`
	MfaType  string `json:"mfaType"`
}

func handleCreateUserMfaV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", ErrorInvalidInput)
		return
	}

	var input handleCreateUserMfaV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("creating mfa of type[%s] for user[%s]", input.MfaType, session.UserId))

	user := models.User{Id: &session.UserId}
	if err := user.LoadByIdV1(models.DatabaseConnection{Db: db}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to retrieve user[%s]: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve user", ErrorDatabaseIssue)
		return
	}

	if !user.ValidatePassword(input.Password) {
		log(common.LogLevelError, fmt.Sprintf("user[%s] entered the wrong password", user.GetId()))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to validate user's current password", ErrorInvalidCredentials)
		return
	}

	switch input.MfaType {
	case models.MfaTypeTotp:
		totpSeed, err := auth.CreateTotpSeed("opsicle", user.Email)
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create totp seed", ErrorCodeIssue)
			return
		}

		userMfa, err := models.CreateUserMfaV1(models.CreateUserMfaV1Opts{
			Db: db,

			UserId: session.UserId,
			Secret: &totpSeed,
			Type:   models.MfaTypeTotp,
		})
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create user totp mfa", ErrorDatabaseIssue)
			return
		}

		userMfa.UserEmail = &user.Email

		audit.Log(audit.LogEntry{
			EntityId:     session.UserId,
			EntityType:   audit.UserEntity,
			Verb:         audit.Create,
			ResourceId:   userMfa.Id,
			ResourceType: audit.UserMfaResource,
			Status:       audit.Success,
			SrcIp:        &session.SourceIp,
			SrcUa:        &session.UserAgent,
			DstHost:      &r.Host,
		})
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", userMfa)
	default:
		common.SendHttpFailResponse(w, r, http.StatusNotFound, "failed to recognise type of mfa", ErrorUnrecognisedMfaType)
		return
	}
}

type handleVerifyUserMfaV1Input struct {
	Value string `json:"value"`
}

type handleVerifyUserMfaV1Output struct {
	Id     string `json:"id"`
	Type   string `json:"type"`
	UserId string `json:"userId"`
}

func handleVerifyUserMfaV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)

	vars := mux.Vars(r)
	mfaId := vars["mfaId"]

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", ErrorInvalidInput)
		return
	}

	var input handleVerifyUserMfaV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("verifying mfa[%s] for user[%s]", mfaId, session.UserId))

	userMfa, err := models.GetUserMfaV1(models.GetUserMfaV1Opts{
		Db: db,
		Id: mfaId,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to get mfa[%s] for user[%s]: %s", mfaId, session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to get user mfa", ErrorDatabaseIssue)
		return
	}

	switch userMfa.Type {
	case models.MfaTypeTotp:
		isValid, err := auth.ValidateTotpToken(*userMfa.Secret, input.Value)
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to validate provided totp token", ErrorTotpInvalid)
			return
		} else if !isValid {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "provided totp token is not valid", ErrorTotpInvalid)
			return
		}
		if err := models.VerifyUserMfaV1(models.VerifyUserMfaV1Opts{
			Db: db,
			Id: mfaId,
		}); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to verify mfa", ErrorDatabaseIssue)
			return
		}
		audit.Log(audit.LogEntry{
			EntityId:     session.UserId,
			EntityType:   audit.UserEntity,
			Verb:         audit.Verify,
			ResourceId:   mfaId,
			ResourceType: audit.UserMfaResource,
			Status:       audit.Success,
			SrcIp:        &session.SourceIp,
			SrcUa:        &session.UserAgent,
			DstHost:      &r.Host,
		})
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleVerifyUserMfaV1Output{
			Id:     userMfa.Id,
			Type:   userMfa.Type,
			UserId: userMfa.UserId,
		})
	}

}

func handleListUserMfasV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
	log(common.LogLevelDebug, fmt.Sprintf("retrieving user[%s]'s available mfas", session.UserId))

	userMfas, err := models.ListUserMfasV1(models.ListUserMfasV1Opts{
		Db:     db,
		UserId: &session.UserId,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to list user[%s] mfas: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to list user mfas", err)
		return
	}
	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.List,
		ResourceType: audit.UserMfaResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", userMfas.GetRedacted())
}

type handleListUserOrgInvitationsV1Output struct {
	Invitations []handleListUserOrgInvitationsV1OutputOrgInvite `json:"invitations"`
}

type handleListUserOrgInvitationsV1OutputOrgInvite struct {
	Id           string    `json:"id"`
	InvitedAt    time.Time `json:"invitedAt"`
	InviterId    string    `json:"inviterId"`
	InviterEmail string    `json:"inviterEmail"`
	JoinCode     string    `json:"joinCode"`
	OrgCode      string    `json:"orgCode"`
	OrgId        string    `json:"orgId"`
	OrgName      string    `json:"orgName"`
}

func handleListUserOrgInvitationsV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
	log(common.LogLevelDebug, fmt.Sprintf("retrieving org invitations for user[%s]", session.UserId))

	listOrgInvitations, err := models.ListOrgInvitationsV1(models.ListOrgInvitationsV1Opts{
		Db:     db,
		UserId: &session.UserId,
	})
	if err != nil {
		if !errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve user", err)
			return
		}
	}
	output := handleListUserOrgInvitationsV1Output{Invitations: []handleListUserOrgInvitationsV1OutputOrgInvite{}}
	for _, orgInvitation := range listOrgInvitations {
		output.Invitations = append(
			output.Invitations,
			handleListUserOrgInvitationsV1OutputOrgInvite{
				Id:           orgInvitation.Id,
				InvitedAt:    orgInvitation.CreatedAt,
				InviterId:    orgInvitation.InviterId,
				InviterEmail: *orgInvitation.InviterEmail,
				JoinCode:     orgInvitation.JoinCode,
				OrgId:        orgInvitation.OrgId,
				OrgCode:      *orgInvitation.OrgCode,
				OrgName:      *orgInvitation.OrgName,
			},
		)
	}
	audit.Log(audit.LogEntry{
		EntityId:     session.UserId,
		EntityType:   audit.UserEntity,
		Verb:         audit.List,
		ResourceType: audit.OrgUserInvitationResource,
		Status:       audit.Success,
		SrcIp:        &session.SourceIp,
		SrcUa:        &session.UserAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}

type handleListUserMfaTypesV1Response []handleListUserMfaTypesV1ResponseType

type handleListUserMfaTypesV1ResponseType struct {
	Description string `json:"description"`
	Label       string `json:"label"`
	Value       string `json:"value"`
}

func handleListUserMfaTypesV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelDebug, "list of all available user mfa types requested")

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleListUserMfaTypesV1Response{
		{
			Value:       models.MfaTypeTotp,
			Label:       "TOTP Token",
			Description: "A time-based one-time-password (via authenticator app)",
		},
	})
}

func handleVerifyUserV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelDebug, "this endpoint verifies a user")
	vars := mux.Vars(r)
	verificationCode := vars["verificationCode"]
	user := models.User{}
	if err := user.VerifyV1(models.VerifyUserV1Opts{
		Db:               db,
		VerificationCode: verificationCode,
		UserAgent:        r.UserAgent(),
		IpAddress:        r.RemoteAddr,
	}); err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to verify user: %s", err))
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to verify user", ErrorInvalidVerificationCode)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify user", ErrorInvalidVerificationCode)
		return
	}
	userAgent := r.UserAgent()
	audit.Log(audit.LogEntry{
		EntityId:     user.GetId(),
		EntityType:   audit.UserEntity,
		Verb:         audit.Verify,
		ResourceType: audit.UserEmailVerificationCodeResource,
		Status:       audit.Success,
		SrcIp:        &r.RemoteAddr,
		SrcUa:        &userAgent,
		DstHost:      &r.Host,
	})
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", user)
}

type handleUpdateUserPasswordV1Output struct {
	IsSuccessful bool `json:"isSuccessful"`
}

type handleUpdateUserPasswordV1Input struct {
	CurrentPassword  *string `json:"currentPassword"`
	Email            *string `json:"email"`
	NewPassword      *string `json:"newPassword"`
	VerificationCode *string `json:"verificationCode"`
}

func handleUpdateUserPasswordV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelDebug, "this endpoint updates a user's password")
	userAgent := r.UserAgent()

	isUserLoggedIn := false
	authorizationHeader := r.Header.Get("Authorization")
	var sessionInfo *models.Session
	if strings.Index(authorizationHeader, "Bearer ") == 0 {
		authorizationToken := strings.ReplaceAll(authorizationHeader, "Bearer ", "")
		log(common.LogLevelDebug, "retrieved an authorization token successfully")
		var err error
		sessionInfo, err = models.GetSessionV1(models.GetSessionV1Opts{
			BearerToken: authorizationToken,
			CachePrefix: sessionCachePrefix,
		})
		if err == nil {
			log(common.LogLevelDebug, fmt.Sprintf("resetting password for user[%s]", sessionInfo.UserId))
			isUserLoggedIn = true
		}
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", ErrorInvalidInput)
		return
	}
	if len(bodyData) == 0 {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get body data", ErrorInvalidInput)
		return
	}
	var input handleUpdateUserPasswordV1Input
	if err := json.Unmarshal(bodyData, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse body data", ErrorInvalidInput)
		return
	}

	isKnownUserChangePasswordFlow := isUserLoggedIn && sessionInfo != nil
	isAnonUserTriggeringPasswodReset := !isUserLoggedIn && input.Email != nil
	isAnonUserVerifyingIdentity := !isUserLoggedIn && input.VerificationCode != nil && input.NewPassword != nil

	switch true {
	case isKnownUserChangePasswordFlow:
		log(common.LogLevelDebug, fmt.Sprintf("user[%s].updatePassword: changing password for user", sessionInfo.UserId))
		// get user from session info
		if sessionInfo.ExpiresAt.Before(time.Now()) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "password change failed", ErrorAuthRequired)
			return
		}

		log(common.LogLevelDebug, fmt.Sprintf("user[%s].updatePassword: validating passwords for password update", sessionInfo.UserId))
		if _, err := auth.IsPasswordValid(*input.CurrentPassword); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid current password", ErrorInvalidInput)
			return
		} else if _, err := auth.IsPasswordValid(*input.NewPassword); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid new password", ErrorInvalidInput)
			return
		}
		currentPassword := *input.CurrentPassword
		newPassword := *input.NewPassword

		userId := sessionInfo.UserId
		log(common.LogLevelDebug, fmt.Sprintf("user[%s].updatePassword: retrieving user details", userId))
		user := models.User{Id: &userId}
		if err := user.LoadByIdV1(models.DatabaseConnection{Db: db}); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get user", ErrorDatabaseIssue)
			return
		}

		log(common.LogLevelDebug, fmt.Sprintf("user[%s].updatePassword: validating current password", userId))
		if !auth.ValidatePassword(currentPassword, *user.PasswordHash) {
			log(common.LogLevelError, fmt.Sprintf("current password verification failed: %s", err))
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed password verification", ErrorInvalidCredentials)
			return
		}

		log(common.LogLevelDebug, fmt.Sprintf("user[%s].updatePassword: updating password", userId))
		if err := user.UpdatePasswordV1(models.UpdateUserPasswordV1Input{
			Db:          db,
			NewPassword: newPassword,
		}); err != nil {
			log(common.LogLevelError, fmt.Sprintf("password update failed: %s", err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed password update", ErrorDatabaseIssue)
			return
		}
		audit.Log(audit.LogEntry{
			EntityId:     userId,
			EntityType:   audit.UserEntity,
			Verb:         audit.Update,
			ResourceId:   userId,
			ResourceType: audit.UserPasswordResource,
			Data:         map[string]any{"authenticated": true},
			Status:       audit.Success,
			SrcIp:        &r.RemoteAddr,
			SrcUa:        &userAgent,
			DstHost:      &r.Host,
		})
		log(common.LogLevelDebug, fmt.Sprintf("user[%s] successfully changed their password", userId))
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleUpdateUserPasswordV1Output{
			IsSuccessful: true,
		})

	case isAnonUserTriggeringPasswodReset:
		userEmail := *input.Email
		log(common.LogLevelDebug, fmt.Sprintf("email[%s].forgotPassword: resetting password for user", userEmail))
		_, err := auth.IsEmailValid(userEmail)
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to receive user email", ErrorInvalidInput)
			return
		}

		user := models.User{Email: userEmail}
		if err := user.LoadByEmailV1(models.DatabaseConnection{Db: db}); err != nil {
			if errors.Is(err, models.ErrorNotFound) {
				// we send a success response because we don't want to alert the user if the email is
				// incorrect
				common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleUpdateUserPasswordV1Output{
					IsSuccessful: true,
				})
				return
			}
			log(common.LogLevelError, fmt.Sprintf("email[%s].forgotPassword: failed to retrieve user: %s", userEmail, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to get user", ErrorDatabaseIssue)
			return
		}

		log(common.LogLevelDebug, fmt.Sprintf("user[%s].forgotPassword: sending verification code to their email", *user.Id))
		verificationCode, err := common.GenerateRandomString(32)
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to create password reset verification code: %s", err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create password reset verification code", ErrorDatabaseIssue)
			return
		}
		passwordResetId, err := models.CreateUserPasswordResetV1(models.CreateUserPasswordResetV1Input{
			Db:               db,
			UserId:           *user.Id,
			IpAddress:        r.RemoteAddr,
			UserAgent:        r.UserAgent(),
			VerificationCode: verificationCode,
		})
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("user[%s].forgotPassword: failed to create password reset database item: %s", *user.Id, err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create password reset item", ErrorDatabaseIssue)
			return
		}
		log(common.LogLevelDebug, fmt.Sprintf("user[%s].forgotPassword: created password reset with id[%s]", *user.Id, passwordResetId))

		if smtpConfig.IsSet() {
			remoteAddr := r.RemoteAddr
			userAgent := r.UserAgent()
			opsicleCatMimeType, opsicleCatData := images.GetOpsicleCat()
			if err := email.SendSmtp(email.SendSmtpOpts{
				ServiceLogs: *serviceLogs,
				To: []email.User{
					{
						Address: user.Email,
					},
				},
				Sender: smtpConfig.Sender,
				Message: email.Message{
					Title: "Did you try to reset your password?",
					Body: templates.GetPasswordResetMessage(
						verificationCode,
						remoteAddr,
						userAgent,
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
				log(common.LogLevelWarn, fmt.Sprintf("failed to send email, send user their verification code[%s] manually", verificationCode))
			}
		}
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleUpdateUserPasswordV1Output{
			IsSuccessful: true,
		})

	case isAnonUserVerifyingIdentity:
		verificationCode := *input.VerificationCode
		newPassword := *input.NewPassword

		log(common.LogLevelDebug, "validating incoming password")
		if _, err := auth.IsPasswordValid(newPassword); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid new password", ErrorInvalidInput)
			return
		}

		log(common.LogLevelDebug, "retrieving user password reset attempt")
		passwordReset, err := models.GetUserPasswordResetV1(models.GetUserPasswordResetV1Input{
			Db: db,

			VerificationCode: verificationCode,
		})
		if err != nil {
			if errors.Is(err, models.ErrorNotFound) {
				common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid verification code", ErrorInvalidInput)
				return
			}
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to identify password reset attempt", ErrorDatabaseIssue)
			return
		}

		log(common.LogLevelDebug, fmt.Sprintf("identified request as user[%s]'s passwordReset attempt[%s]", passwordReset.UserId, passwordReset.Id))
		user := models.User{Id: &passwordReset.Id}
		if err := user.LoadByIdV1(models.DatabaseConnection{Db: db}); err != nil {
			if errors.Is(err, models.ErrorNotFound) {
				common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid user referenced", ErrorInvalidInput)
				return
			}
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve user", ErrorDatabaseIssue)
			return
		}

		log(common.LogLevelDebug, fmt.Sprintf("user[%s].updatePassword: updating password", user.GetId()))
		if err := user.UpdatePasswordV1(models.UpdateUserPasswordV1Input{
			Db:          db,
			NewPassword: newPassword,
		}); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed password update", ErrorDatabaseIssue)
			return
		}

		audit.Log(audit.LogEntry{
			EntityId:     user.GetId(),
			EntityType:   audit.UserEntity,
			Verb:         audit.Update,
			ResourceId:   user.GetId(),
			ResourceType: audit.UserPasswordResource,
			Data:         map[string]any{"authenticated": false},
			Status:       audit.Success,
			SrcIp:        &r.RemoteAddr,
			SrcUa:        &userAgent,
			DstHost:      &r.Host,
		})
		log(common.LogLevelDebug, fmt.Sprintf("user[%s] successfully changed their password", user.GetId()))
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleUpdateUserPasswordV1Output{
			IsSuccessful: true,
		})

		if err := models.SetUserPasswordResetToSuccessV1(models.SetUserPasswordResetToSuccessV1Input{
			Db: db,
			Id: passwordReset.Id,
		}); err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to update password reset attempt to successful: %s", err))
		}

	default:
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "bad request", ErrorInvalidInput)
		return
	}

}
