package controller

import (
	"context"
	"net/http"
	"opsicle/internal/common"
)

type requestContextType string

const authRequestContext requestContextType = "auth"

type identity struct {
	// OrganizationId is the ID of the current caller's organization
	OrganizationId string `json:"organizationId"`

	// OrganizationRoleId is the ID of the current caller's role within
	// the organization identified by OrganizationId
	OrganizationRoleId string `json:"organizationRoleId"`
}

func getRouteAuther(serviceLogs chan<- common.ServiceLog) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "auth middleware executing")
			// bearerToken := r.Header.Get("Authorization")
			// TODO: check which organisation they're from
			serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "auth middleware executing")
			identityInstance := identity{
				OrganizationId:     "todo:id",
				OrganizationRoleId: "todo:roleId",
			}
			authContext := context.WithValue(r.Context(), authRequestContext, identityInstance)
			next.ServeHTTP(w, r.WithContext(authContext))
		})
	}
}
