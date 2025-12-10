package controller

import (
	"context"
	"fmt"
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"
	"opsicle/internal/types"
	"strings"
)

const userAuthRequestContext common.HttpContextKey = "controller-auth"
const internalAuthRequestContext common.HttpContextKey = "internal-auth"

type apiIdentity struct {
	// Status is the status of the API key
	Status string `json:"status"`

	// Value is the API key
	Value string `json:"value"`
}

type userIdentity struct {
	// SourceIp is the IP address that the request came from
	SourceIp string `json:"sourceIp"`

	// UserAgent is the user agent of the request
	UserAgent string `json:"userAgent"`

	// UserId is the ID of the current caller
	UserId string `json:"userId"`

	// Username is the email of the current caller
	Username string `json:"username"`
}

func getRouteAuther(serviceLogs chan<- common.ServiceLog) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
			serviceLogs <- common.ServiceLogf(common.LogLevelTrace, "auth middleware is executing")
			authorizationHeader := r.Header.Get("Authorization")
			if strings.Index(authorizationHeader, "Bearer ") != 0 {
				common.SendHttpFailResponse(w, r, http.StatusUnauthorized, "failed to receive an authorization header", types.ErrorAuthRequired)
				return
			}
			authorizationToken := strings.ReplaceAll(authorizationHeader, "Bearer ", "")
			sessionInfo, err := models.GetSessionV1(models.GetSessionV1Opts{
				BearerToken: authorizationToken,
				CachePrefix: sessionCachePrefix,
			})
			if err != nil {
				common.SendHttpFailResponse(w, r, http.StatusUnauthorized, "failed to retrieve session", types.ErrorAuthRequired)
				return
			}
			log(common.LogLevelInfo, fmt.Sprintf("processing request from user[%s]", sessionInfo.UserId))
			identityInstance := userIdentity{
				SourceIp:  r.RemoteAddr,
				UserId:    sessionInfo.UserId,
				Username:  sessionInfo.Username,
				UserAgent: r.UserAgent(),
			}
			authContext := context.WithValue(r.Context(), userAuthRequestContext, identityInstance)
			next.ServeHTTP(w, r.WithContext(authContext))
		})
	}
}

func getInternalRouteAuther(apiKeys []string, serviceLogs chan<- common.ServiceLog) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
			serviceLogs <- common.ServiceLogf(common.LogLevelTrace, "internal auth middleware is executing")
			apiKey := r.Header.Get("x-api-key")

			var apiKeyIndex int
			var apiKeysCount = len(apiKeys)
			for apiKeyIndex = 0; apiKeyIndex < apiKeysCount; apiKeyIndex++ {
				if apiKeys[apiKeyIndex] == apiKey {
					break
				}
			}
			if apiKeyIndex == apiKeysCount {
				log(common.LogLevelWarn, fmt.Sprintf("api key validation failed with key '%s'", apiKey))
				common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to receive a valid api key", types.ErrorAuthRequired)
				return
			}
			apiIdentityInstance := apiIdentity{
				Status: "ok",
				Value:  apiKeys[apiKeyIndex],
			}
			if apiKeyIndex != 0 {
				apiIdentityInstance.Status = "deprecated"
			}
			w.Header().Set("x-api-key-status", apiIdentityInstance.Status)
			authContext := context.WithValue(r.Context(), internalAuthRequestContext, apiIdentityInstance)
			next.ServeHTTP(w, r.WithContext(authContext))
		})
	}
}
