package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"opsicle/internal/auth"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func registerSessionRoutes(opts RouteRegistrationOpts) {
	v1 := opts.Router.PathPrefix("/v1/session").Subrouter()

	v1.HandleFunc("", handleCreateSessionV1).Methods(http.MethodPost)
	v1.HandleFunc("", handleGetSessionV1).Methods(http.MethodGet)
	v1.HandleFunc("", handleDeleteSessionV1).Methods(http.MethodDelete)
	v1.HandleFunc("/mfa/{loginId}", handleStartSessionWithMfaV1).Methods(http.MethodPost)
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

	// MfaToken is the MFA token of a user (if applicable)
	MfaToken *string `json:"mfaToken"`

	// MfaType is the type of the MFA method (if applicable)
	MfaType *string `json:"mfaType"`
}

type handleCreateSessionV1MfaRequiredResponse struct {
	LoginId string `json:"loginId"`
	MfaType string `json:"mfaType"`
}

type handleCreateSessionV1Output struct {
	SessionId    string `json:"sessionId"`
	SessionToken string `json:"sessionToken"`
}

// handleCreateSessionV1 godoc
// @Summary      Creates a session for the user credentials specified in the body
// @Description  This endpoint creates a session for the user
// @Tags         controller-service
// @Accept       json
// @Produce      json
// @Param        request body handleCreateSessionV1Input true "User credentials"
// @Success      200 {object} commonHttpResponse "ok"
// @Failure      401 {object} commonHttpResponse "unauthorized"
// @Failure      403 {object} commonHttpResponse "forbidden"
// @Failure      500 {object} commonHttpResponse "internal server error"
// @Router       /api/v1/session [post]
func handleCreateSessionV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, "successfully read body into bytes")
	var input handleCreateSessionV1Input
	if err := json.Unmarshal(requestBody, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, "successfully parsed body into expected input class")

	if input.Email == "" {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to receive a valid org email", ErrorInvalidInput)
		return
	}
	userInstance, err := models.GetUserV1(models.GetUserV1Opts{
		Db:    db,
		Email: &input.Email,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to receive a valid user", ErrorDatabaseIssue)
		return
	}

	userInstance.Password = &input.Password
	if !userInstance.ValidatePassword() {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to create session", ErrorInvalidCredentials)
		return
	} else if !userInstance.IsEmailVerified {
		common.SendHttpFailResponse(w, r, http.StatusLocked, "failed to create session", ErrorEmailUnverified)
		return
	} else if userInstance.IsDisabled || userInstance.IsDeleted {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create session", ErrorAccountSuspended)
		return
	}

	userMfas, err := models.ListUserMfasV1(models.ListUserMfasV1Opts{
		Db:     db,
		UserId: userInstance.Id,
	})
	if err != nil {
		log(common.LogLevelError, fmt.Sprintf("failed to list user's mfas: %s", err))
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to list user's mfas", ErrorDatabaseIssue)
		return
	} else if len(userMfas) > 0 {
		userLoginId, err := models.CreateUserLoginV1(models.CreateUserLoginV1Input{
			Db:          db,
			UserId:      *userInstance.Id,
			RequiresMfa: true,
		})
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to create a user login attempt: %s", err))
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create user login", ErrorDatabaseIssue)
			return
		}
		log(common.LogLevelDebug, fmt.Sprintf("responding with request for mfa from user[%s]", input.Email))

		common.SendHttpFailResponse(w, r, http.StatusUnauthorized, "provide mfa token", ErrorMfaRequired, handleCreateSessionV1MfaRequiredResponse{
			LoginId: userLoginId,
			MfaType: userMfas[rand.IntN(len(userMfas))].Type,
		})
		return
	}

	userLoginId, err := models.CreateUserLoginV1(models.CreateUserLoginV1Input{
		Db:     db,
		UserId: *userInstance.Id,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create user login entry", err)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("logged user login attempt as login[%s]", userLoginId))

	sessionToken, err := models.CreateSessionV1(models.CreateSessionV1Opts{
		Db:          db,
		CachePrefix: sessionCachePrefix,
		Email:       input.Email,
		IpAddress:   r.RemoteAddr,
		UserAgent:   r.UserAgent(),
		Hostname:    input.Hostname,
		Source:      "api",
		ExpiresIn:   12 * time.Hour,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to create session", err)
		return
	}
	log(common.LogLevelDebug, "successfully issued session token")

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleCreateSessionV1Output{
		SessionId:    sessionToken.SessionId,
		SessionToken: sessionToken.Value,
	})
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
		common.SendHttpFailResponse(w, r, http.StatusUnauthorized, "failed to receive a valid authorization header", ErrorAuthRequired)
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
		common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to receive a valid authorization header", ErrorAuthRequired)
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

type handleStartSessionWithMfaV1Input struct {
	Hostname *string `json:"hostname"`
	MfaType  string  `json:"mfaType"`
	MfaToken string  `json:"mfaToken"`
}

// handleStartSessionWithMfaV1 godoc
// @Summary      Logs the current user out
// @Description  This endpoint deletes the session which the user is currently using
// @Tags         controller-service
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} commonHttpResponse "ok"
// @Failure      400 {object} commonHttpResponse "bad request"
// @Failure      403 {object} commonHttpResponse "forbidden"
// @Router       /api/v1/session/mfa [post]
func handleStartSessionWithMfaV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, "successfully read body into bytes")
	var input handleStartSessionWithMfaV1Input
	if err := json.Unmarshal(requestBody, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, "successfully parsed body into expected input class")
	vars := mux.Vars(r)
	loginId := vars["loginId"]

	userLogin, err := models.GetUserLoginV1(models.GetUserLoginV1Input{
		Db:      db,
		LoginId: loginId,
	})
	if err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get user login", ErrorInvalidInput)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to get user login", ErrorInvalidInput)
		return
	}

	user, err := models.GetUserV1(models.GetUserV1Opts{
		Db: db,

		Id: &userLogin.UserId,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get user", ErrorGeneric)
		return
	}

	userMfas, err := models.ListUserMfasV1(models.ListUserMfasV1Opts{
		Db:     db,
		UserId: user.Id,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to get user mfas", ErrorGeneric)
		return
	}

	found := false
	for _, userMfa := range userMfas {
		if userMfa.Type == input.MfaType {
			switch userMfa.Type {
			case models.MfaTypeTotp:
				valid, err := auth.ValidateTotpToken(*userMfa.Secret, input.MfaToken)
				if err != nil || !valid {
					log(common.LogLevelError, fmt.Sprintf("failed to validate mfa for user[%s] of type[%s]", *user.Id, userMfa.Type))
					common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to authenticate user mfa", ErrorMfaTokenInvalid)
					return
				}
				if err := models.SetUserLoginMfaSucceededV1(models.SetUserLoginMfaSucceededV1Input{
					Db: db,
					Id: loginId,
				}); err != nil {
					log(common.LogLevelError, fmt.Sprintf("failed to set mfa status for user[%s]'s login[%s] to truthy", *user.Id, userLogin.Id))
					common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to authenticate user mfa", ErrorDatabaseIssue)
					return
				}
				createSessionInput := models.CreateSessionV1Opts{
					Db:          db,
					CachePrefix: sessionCachePrefix,

					Email: user.Email,

					IpAddress: r.RemoteAddr,
					UserAgent: r.UserAgent(),
					Source:    "api",
					ExpiresIn: 12 * time.Hour,
				}
				if input.Hostname != nil {
					createSessionInput.Hostname = *input.Hostname
				}
				sessionToken, err := models.CreateSessionV1(createSessionInput)
				if err != nil {
					common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to create session", err)
					return
				}
				log(common.LogLevelDebug, "successfully issued session token")

				common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", sessionToken)

			default:
				log(common.LogLevelError, fmt.Sprintf("failed to identify mfa for user[%s] of type[%s]", *user.Id, userMfa.Type))
				common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to authenticate user mfa", ErrorGeneric)
				return
			}
			found = true
			break
		}
	}
	if !found {
		log(common.LogLevelError, fmt.Sprintf("no mfa for user[%s] of type[%s] was found", *user.Id, input.MfaType))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to authenticate user mfa", ErrorGeneric)
		return
	}

}
