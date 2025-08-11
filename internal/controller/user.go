package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/auth"
	"opsicle/internal/common"
	"opsicle/internal/common/images"
	"opsicle/internal/controller/models"
	"opsicle/internal/controller/templates"
	"opsicle/internal/email"

	"github.com/gorilla/mux"
)

func registerUserRoutes(opts RouteRegistrationOpts) {
	// requiresAuth := getRouteAuther(opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1/users").Subrouter()

	v1.HandleFunc("", handleCreateUserV1).Methods(http.MethodPost)

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

	if !auth.IsEmailValid(requestData.Email) {
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
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create user", err)
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
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to verify user", err)
		return
	}
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", userInstance)
}
