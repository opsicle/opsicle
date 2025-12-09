package coordinator

import (
	"context"
	"fmt"
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/types"
	"strings"
)

const authRequestContext common.HttpContextKey = "coordinator-auth"

type identity struct {
	// SourceIp is the IP address that the request came from
	SourceIp string `json:"sourceIp"`

	// UserAgent is the user agent of the request
	UserAgent string `json:"userAgent"`

	// OrgId is the ID of the current caller
	OrgId string `json:"orgId"`
}

func getRouteAuther(serviceLogs chan<- common.ServiceLog) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
			serviceLogs <- common.ServiceLogf(common.LogLevelTrace, "auth middleware is executing")
			authorizationHeader := r.Header.Get("Authorization")
			if strings.Index(authorizationHeader, "Bearer ") != 0 {
				common.SendHttpFailResponse(w, r, http.StatusUnauthorized, "failed to receive an authorization header", types.ErrorAuthRequired)
				return
			}
			authorizationToken := strings.ReplaceAll(authorizationHeader, "Bearer ", "")
			fmt.Println(authorizationToken)
			// sessionInfo, err := models.GetSessionV1(models.GetSessionV1Opts{
			// 	BearerToken: authorizationToken,
			// 	CachePrefix: sessionCachePrefix,
			// })
			// if err != nil {
			// 	common.SendHttpFailResponse(w, r, http.StatusUnauthorized, "failed to retrieve session", types.ErrorAuthRequired)
			// 	return
			// }
			identityInstance := identity{
				SourceIp:  r.RemoteAddr,
				OrgId:     "TODO",
				UserAgent: r.UserAgent(),
			}
			authContext := context.WithValue(r.Context(), authRequestContext, identityInstance)
			next.ServeHTTP(w, r.WithContext(authContext))
		})
	}
}
