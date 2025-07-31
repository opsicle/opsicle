package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"
)

func registerUserRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1/users").Subrouter()

	v1.HandleFunc("", handleCreateUserV1).Methods(http.MethodPost)
	v1.Handle("", requiresAuth(http.HandlerFunc(handleListUsersV1))).Methods(http.MethodGet)
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

	if _, err := mail.ParseAddress(requestData.Email); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid email address", err)
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

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleCreateUserV1Output{
		Id:    *user.Id,
		Email: user.Email,
	})
}

func handleListUsersV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelDebug, "this endpoint retrieves users from the current user's organisation")
	session := r.Context().Value(authRequestContext).(identity)

	if session.OrganizationId == nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "user is not logged into an organisation", nil)
		return
	}

	users, err := models.ListUsersV1(models.ListUsersV1Opts{
		Db:      db,
		OrgCode: *session.OrganizationCode,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "not ok", err)
		return
	}

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", users)
}
