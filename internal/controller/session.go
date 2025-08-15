package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"
	"strings"
	"time"
)

func registerSessionRoutes(opts RouteRegistrationOpts) {
	v1 := opts.Router.PathPrefix("/v1/session").Subrouter()

	v1.HandleFunc("", handleCreateSessionV1).Methods(http.MethodPost)
	v1.HandleFunc("", handleGetSessionV1).Methods(http.MethodGet)
	v1.HandleFunc("", handleDeleteSessionV1).Methods(http.MethodDelete)
}

type handleCreateSessionV1Input struct {
	// Email is the user's email address
	Email string `json:"email"`

	// OrgCode is the user's organisation code
	OrgCode *string `json:"orgCode"`

	// Password is the user's password
	Password string `json:"password"`

	// Hostname is the user's machine's hostname
	Hostname string `json:"hostname"`
}

// handleCreateSessionV1 godoc
// @Summary      Creates a session for the user credentials specified in the body
// @Description  This endpoint creates a session for the user
// @Tags         controller-service
// @Accept       json
// @Produce      json
// @Param        request body handleCreateSessionV1Input true "User credentials"
// @Success      200 {object} commonHttpResponse "ok"
// @Failure      403 {object} commonHttpResponse "forbidden"
// @Failure      500 {object} commonHttpResponse "internal server error"
// @Router       /api/v1/session [post]
func handleCreateSessionV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", nil)
		return
	}
	log(common.LogLevelDebug, "successfully read body into bytes")
	var input handleCreateSessionV1Input
	if err := json.Unmarshal(requestBody, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", nil)
		return
	}
	log(common.LogLevelDebug, "successfully parsed body into expected input class")

	if input.Email == "" {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to receive a valid org email", nil)
		return
	}

	sessionToken, err := models.CreateSessionV1(models.CreateSessionV1Opts{
		Db:          db,
		CachePrefix: sessionCachePrefix,

		Email:    input.Email,
		OrgCode:  input.OrgCode,
		Password: input.Password,

		IpAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		Hostname:  input.Hostname,
		Source:    "api",
		ExpiresIn: 12 * time.Hour,
	})
	if err != nil {
		if errors.Is(err, models.ErrorCredentialsAuthenticationFailed) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to create session", err)
			return
		} else if errors.Is(err, models.ErrorUserEmailNotVerified) {
			common.SendHttpFailResponse(w, r, http.StatusLocked, "failed to create session", err)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create session", err)
		return
	}
	log(common.LogLevelDebug, "successfully issued session token")

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", sessionToken)
}

// handleGetSessionV1 godoc
// @Summary      Retrieves the user's current session
// @Description  This endpoint returns information about the user's current session
// @Tags         controller-service
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} commonHttpResponse "ok"
// @Failure      403 {object} commonHttpResponse "forbidden"
// @Router       /api/v1/session [get]
func handleGetSessionV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)

	authorizationHeader := r.Header.Get("Authorization")
	if strings.Index(authorizationHeader, "Bearer ") != 0 {
		common.SendHttpFailResponse(w, r, http.StatusUnauthorized, "failed to receive a valid authorization header")
		return
	}
	authorizationToken := strings.ReplaceAll(authorizationHeader, "Bearer ", "")
	log(common.LogLevelDebug, "retrieved an authorizationToken successfully")

	sessionInfo, err := models.GetSessionV1(models.GetSessionV1Opts{
		BearerToken: authorizationToken,
		CachePrefix: sessionCachePrefix,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusUnauthorized, "failed to retrieve session details", ErrorAuthRequired)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("session[%s] is valid and has %s time left", sessionInfo.Id, sessionInfo.TimeLeft))

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", sessionInfo)
}

type handleDeleteSessionV1Output struct {
	SessionId    string `json:"sessionId"`
	IsSuccessful bool   `json:"isSuccessful"`
}

// handleDeleteSessionV1 godoc
// @Summary      Logs the current user out
// @Description  This endpoint deletes the session which the user is currently using
// @Tags         controller-service
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} commonHttpResponse "ok"
// @Failure      403 {object} commonHttpResponse "forbidden"
// @Router       /api/v1/session [delete]
func handleDeleteSessionV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	authorizationHeader := r.Header.Get("Authorization")
	if strings.Index(authorizationHeader, "Bearer ") != 0 {
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to receive a valid authorization header", nil)
		return
	}
	authorizationToken := strings.ReplaceAll(authorizationHeader, "Bearer ", "")
	log(common.LogLevelDebug, "retrieved an authorization token successfully")

	sessionId, err := models.DeleteSessionV1(models.DeleteSessionV1Opts{
		BearerToken: authorizationToken,
		CachePrefix: sessionCachePrefix,
	})
	if err != nil {
		log(common.LogLevelWarn, fmt.Sprintf("failed to delete session: %s", err))
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleDeleteSessionV1Output{
			SessionId:    "",
			IsSuccessful: false,
		})
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("session[%s] has been deleted", sessionId))
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleDeleteSessionV1Output{
		SessionId:    sessionId,
		IsSuccessful: true,
	})
}
