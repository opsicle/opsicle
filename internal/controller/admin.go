package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/auth"
	"opsicle/internal/common"
	"strings"

	"github.com/google/uuid"
)

func registerAdminRoutes(opts RouteRegistrationOpts, adminToken string) {
	requiresAuth := getAdminRouteAuther(adminToken, opts.ServiceLogs)
	v1 := opts.Router.PathPrefix("/v1").Subrouter()
	v1.Use(requiresAuth)

	v1.HandleFunc("/init", initHandlerV1).Methods(http.MethodPost)
}

type initHandlerV1Input struct {
	// Email is the root user's email address
	Email string `json:"email"`
	// Password is the root user's password
	Password string `json:"password"`
}

func initHandlerV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelInfo, "this endpoint initialises the server")
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", nil)
		return
	}
	var input initHandlerV1Input
	if err := json.Unmarshal(requestBody, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", nil)
		return
	}
	userUuid := uuid.New().String()
	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to hash password", nil)
		return
	}

	stmt, err := db.Prepare("INSERT INTO users(id, email, password_hash, type) VALUES (?, ?, ?, ?)")
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to prepare insert statement", err)
		return
	}

	res, err := stmt.Exec(userUuid, input.Email, passwordHash, "sysadmin")
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to execute insert statement", err)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err == nil {
		log(common.LogLevelInfo, fmt.Sprintf("%v row(s) created in the users table", rowsAffected))
	}

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "user created")
}

func getAdminRouteAuther(adminToken string, serviceLogs chan<- common.ServiceLog) func(http.Handler) http.Handler {
	if adminToken == "" { // just incase someone disables the main disabling when adminToken is ""
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "admin endpoint is disabled but an attempt was made to an admin endpoint")
				common.SendHttpFailResponse(w, r, http.StatusForbidden, "forbidden", fmt.Errorf("disabled"))
			})
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorizationHeader := r.Header.Get("Authorization")
			authorizationParts := strings.SplitN(authorizationHeader, " ", 2)
			if len(authorizationParts) != 2 {
				serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "expected 2 parts to admin route authorization header but found %v", len(authorizationParts))
				common.SendHttpFailResponse(w, r, http.StatusForbidden, "forbidden", fmt.Errorf("wrong format"))
				return
			}
			if authorizationParts[0] != "Bearer" {
				serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "admin route authorization header looks weird, possibly not a bearer token")
				common.SendHttpFailResponse(w, r, http.StatusForbidden, "forbidden", fmt.Errorf("wrong format"))
				return
			}
			receivedToken := authorizationParts[1]
			if receivedToken != adminToken {
				serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "a wrong admin token was supplied")
				common.SendHttpFailResponse(w, r, http.StatusForbidden, "forbidden", fmt.Errorf("invalid token"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
