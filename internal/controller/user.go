package controller

import (
	"fmt"
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"
)

func registerUserRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1/users").Subrouter()

	v1.Handle("", requiresAuth(http.HandlerFunc(handleListUsersV1))).Methods(http.MethodGet)
}

func handleListUsersV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelDebug, "this endpoint retrieves users from the current user's organisation")

	session := r.Context().Value(authRequestContext).(identity)
	fmt.Printf("received request from user[%s] in org[%s]", session.Username, session.OrganizationCode)

	users, err := models.ListUsersV1(models.ListUsersV1Opts{
		Db:      db,
		OrgCode: session.OrganizationCode,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "not ok", err)
		return
	}

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", users)
}
