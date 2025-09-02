package controller

import (
	"context"
	"fmt"
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"
	"strings"
)

const authRequestContext common.HttpContextKey = "controller-auth"

type identity struct {
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
				common.SendHttpFailResponse(w, r, http.StatusUnauthorized, "failed to receive an authorization header", ErrorAuthRequired)
				return
			}
			authorizationToken := strings.ReplaceAll(authorizationHeader, "Bearer ", "")
			sessionInfo, err := models.GetSessionV1(models.GetSessionV1Opts{
				BearerToken: authorizationToken,
				CachePrefix: sessionCachePrefix,
			})
			if err != nil {
				common.SendHttpFailResponse(w, r, http.StatusUnauthorized, "failed to retrieve session", ErrorAuthRequired)
				return
			}
			log(common.LogLevelInfo, fmt.Sprintf("request from user[%s]", sessionInfo.Username))
			identityInstance := identity{
				SourceIp:  r.RemoteAddr,
				UserId:    sessionInfo.UserId,
				Username:  sessionInfo.Username,
				UserAgent: r.UserAgent(),
			}
			authContext := context.WithValue(r.Context(), authRequestContext, identityInstance)
			next.ServeHTTP(w, r.WithContext(authContext))
		})
	}
}
