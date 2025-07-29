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
	// OrganizationId is the ID of the current caller's organization
	OrganizationId string `json:"organizationId"`

	// OrganizationCode is the code of the current caller's organization
	OrganizationCode string `json:"organizationCode"`

	// UserId is the ID of the current caller
	UserId string `json:"userId"`

	// Username is the email of the current caller
	Username string `json:"username"`
}

func getRouteAuther(serviceLogs chan<- common.ServiceLog) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "auth middleware executing")
			authorizationHeader := r.Header.Get("Authorization")
			if strings.Index(authorizationHeader, "Bearer ") != 0 {
				common.SendHttpFailResponse(w, r, http.StatusForbidden, "failed to receive a valid authorization header", nil)
				return
			}
			authorizationToken := strings.ReplaceAll(authorizationHeader, "Bearer ", "")
			sessionInfo, err := models.GetSessionV1(models.GetSessionV1Opts{
				BearerToken: authorizationToken,
				CachePrefix: sessionCachePrefix,
			})
			if err != nil {
				common.SendHttpFailResponse(w, r, http.StatusUnauthorized, "failed to identify a valid session", err)
				return
			}
			serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "auth middleware executing")

			log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
			log(common.LogLevelDebug, fmt.Sprintf("incoming request from user[%s] from org[%s]", sessionInfo.Username, sessionInfo.OrgCode))
			identityInstance := identity{
				OrganizationCode: sessionInfo.OrgCode,
				OrganizationId:   sessionInfo.OrgId,
				UserId:           sessionInfo.UserId,
				Username:         sessionInfo.Username,
			}
			authContext := context.WithValue(r.Context(), authRequestContext, identityInstance)
			next.ServeHTTP(w, r.WithContext(authContext))
		})
	}
}
