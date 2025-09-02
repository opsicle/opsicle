package controller

import (
	"fmt"
	"net/http"
	"opsicle/internal/common"
	"strings"
)

func registerAdminRoutes(opts RouteRegistrationOpts, adminToken string) {
	requiresAuth := getAdminRouteAuther(adminToken, opts.ServiceLogs)
	v1 := opts.Router.PathPrefix("/v1").Subrouter()
	v1.Use(requiresAuth)
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
