package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/auth"
	"opsicle/internal/common"
	"opsicle/internal/common/images"
	"opsicle/internal/controller/models"
	"opsicle/internal/controller/templates"
	"opsicle/internal/email"
	"strings"

	"github.com/gorilla/mux"
)

func registerUserRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1/users").Subrouter()

	v1.HandleFunc("", handleCreateUserV1).Methods(http.MethodPost)

	v1 = opts.Router.PathPrefix("/v1/user").Subrouter()

	v1.HandleFunc("/password", handleUpdateUserPasswordV1).Methods(http.MethodPatch)
	v1.Handle("/mfa", requiresAuth(http.HandlerFunc(handleCreateUserMfaV1))).Methods(http.MethodPost)
	v1.Handle("/mfa/{mfaId}", requiresAuth(http.HandlerFunc(handleVerifyUserMfaV1))).Methods(http.MethodPost)
	v1.Handle("/mfas", requiresAuth(http.HandlerFunc(handleListUserMfasV1))).Methods(http.MethodGet)
	v1.HandleFunc("/mfas", handleListUserMfaTypesV1).Methods(http.MethodOptions)

	v1 = opts.Router.PathPrefix("/v1/verification").Subrouter()

	v1.HandleFunc("/{verificationCode}", handleVerifyUserV1).Methods(http.MethodGet)
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
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", err)
		return
	}
	var requestData handleCreateUserV1Input
	if err := json.Unmarshal(requestBody, &requestData); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", err)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("processing request to create user[%s]", requestData.Email))

	if _, err := auth.IsEmailValid(requestData.Email); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid email address", err)
		return
	}
	if _, err := auth.IsPasswordValid(requestData.Password); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid password", err)
		return
	}

	if err := models.CreateUserV1(models.CreateUserV1Opts{
		Db: db,

		Email:    requestData.Email,
		Password: requestData.Password,
		Type:     models.TypeUser,
	}); err != nil {
		if errors.Is(err, models.ErrorDuplicateEntry) {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create user, email already exists", ErrorEmailExists)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create user for unexpected reasons", err)
		return
	}

	user, err := models.GetUserV1(models.GetUserV1Opts{
		Db: db,

		Email: &requestData.Email,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve user", err)
		return
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

	user, err := models.GetUserV1(models.GetUserV1Opts{
		Db: db,
		Id: &session.UserId,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to retrieve user[%s]: %s", session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve user", ErrorDatabaseIssue)
		return
	}

	if !auth.ValidatePassword(input.Password, *user.PasswordHash) {
		log(common.LogLevelError, "failed to validate user password")
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
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", userMfas.GetRedacted())

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
	userInstance, err := models.VerifyUserV1(models.VerifyUserV1Opts{
		Db:               db,
		VerificationCode: verificationCode,
		UserAgent:        r.UserAgent(),
		IpAddress:        r.RemoteAddr,
	})
	if err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to verify user", ErrorInvalidVerificationCode)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify user", ErrorInvalidVerificationCode)
		return
	}
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", userInstance)
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

	isUserLoggedIn := false
	authorizationHeader := r.Header.Get("Authorization")
	if strings.Index(authorizationHeader, "Bearer ") == 0 {
		authorizationToken := strings.ReplaceAll(authorizationHeader, "Bearer ", "")
		log(common.LogLevelDebug, "retrieved an authorization token successfully")
		sessionInfo, err := models.GetSessionV1(models.GetSessionV1Opts{
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

	if !isUserLoggedIn {
		if input.Email == nil || *input.Email == "" {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to receive user email", ErrorInvalidInput)
			return
		}

		if input.VerificationCode != nil {
			// check verification code
			// set password to the new password
		}

		userEmail := *input.Email
		log(common.LogLevelDebug, fmt.Sprintf("resetting password for user with email[%s]", email))
		_, err := auth.IsEmailValid(userEmail)
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to receive user email", ErrorInvalidInput)
			return
		}

		user, err := models.GetUserV1(models.GetUserV1Opts{
			Db: db,

			Email: &userEmail,
		})
		if err != nil {
			if errors.Is(err, models.ErrorNotFound) {
				// we send a success response because we don't want to alert the user if the email is
				// incorrect
				common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleUpdateUserPasswordV1Output{
					IsSuccessful: true,
				})
				return
			}
			log(common.LogLevelError, fmt.Sprintf("failed to retrieve user: %w", err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to get user", ErrorDatabaseIssue)
			return
		}

		verificationCode, err := common.GenerateRandomString(32)
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to create password reset verification code: %w", err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create password reset verification code", ErrorDatabaseIssue)
			return
		}
		passwordResetId, err := models.CreateUserPasswordResetV1(models.CreateUserPasswordResetV1Input{
			Db:               db,
			UserId:           *user.Id,
			IpAddress:        r.RemoteAddr,
			UserAgent:        r.UserAgent(),
			VerificationCode: verificationCode,
			Status:           "pending",
		})
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to create password reset database item: %w", err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create password reset item", ErrorDatabaseIssue)
			return
		}
		log(common.LogLevelDebug, fmt.Sprintf("created password reset with id[%s]", passwordResetId))

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
		return
	}

	if input.CurrentPassword == nil || input.NewPassword == nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get passwords", ErrorInvalidInput)
		return
	}

	// compare current password
	// if corerct => update password to the new one

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleUpdateUserPasswordV1Output{
		IsSuccessful: true,
	})
}
