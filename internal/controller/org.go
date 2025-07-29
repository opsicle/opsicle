package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"
)

func registerOrgRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1/org").Subrouter()

	v1.Handle("", requiresAuth(http.HandlerFunc(handleGetCurrentOrgV1))).Methods(http.MethodGet)
	v1.HandleFunc("", handleCreateOrgV1).Methods(http.MethodPost)
}

type handleCreateOrgV1Input struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// handleGetSessionV1 godoc
// @Summary      Creates a session for the user credentials specified in the body
// @Description  This endpoint creates a session for the user
// @Tags         controller-service
// @Accept       json
// @Produce      json
// @Param        request body handleCreateOrgV1Input true "User credentials"
// @Success      200 {object} commonHttpResponse "ok"
// @Failure      403 {object} commonHttpResponse "forbidden"
// @Failure      500 {object} commonHttpResponse "internal server error"
// @Router       /api/v1/session [post]
func handleCreateOrgV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", nil)
		return
	}
	log(common.LogLevelDebug, "successfully read body into bytes")
	var input handleCreateOrgV1Input
	if err := json.Unmarshal(requestBody, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", nil)
		return
	}
	log(common.LogLevelDebug, "successfully parsed body into expected input class")

	if err := models.CreateOrgV1(models.CreateOrgV1Opts{
		Db:   db,
		Code: input.Code,
		Name: input.Name,
	}); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create org", err)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("successfully created org[%s]", input.Code))

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", input.Code)
}

func handleGetCurrentOrgV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
	log(common.LogLevelDebug, fmt.Sprintf("retrieving org[%s]...", session.OrganizationId))
	orgInstance, err := models.GetOrgV1(models.GetOrgV1Opts{
		Db: db,
		Id: &session.OrganizationId,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to retrieve org[%s]", session.OrganizationId), err)
		return
	}
	if _, err := orgInstance.LoadUserCountV1(models.LoadOrgUserCountV1Opts{Db: db}); err != nil {
		log(common.LogLevelWarn, fmt.Sprintf("failed to get user count of org[%s]: %s", session.OrganizationId, err))
	}
	log(common.LogLevelDebug, fmt.Sprintf("successfully retrieved org[%s]", session.OrganizationId))

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", orgInstance)
}
