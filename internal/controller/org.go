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
	v1.Handle("", requiresAuth(http.HandlerFunc(handleCreateOrgV1))).Methods(http.MethodPost)
}

type handleCreateOrgV1Output struct {
	Id   string `json:"id"`
	Code string `json:"code"`
}

type handleCreateOrgV1Input struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// handleCreateOrgV1 godoc
// @Summary      Creates a new organisation
// @Description  Creates a new organisation and assigns the user identified by their token as the administrator of the organisation
// @Tags         controller-service
// @Accept       json
// @Produce      json
// @Param        request body handleCreateOrgV1Input true "User credentials"
// @Success      200 {object} commonHttpResponse "ok"
// @Failure      403 {object} commonHttpResponse "forbidden"
// @Failure      500 {object} commonHttpResponse "internal server error"
// @Router       /api/v1/org [post]
func handleCreateOrgV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)
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

	if len(input.Name) < 4 {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "org name should be longer than 4 characters", nil)
		return
	}
	if len(input.Code) < 4 {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "org code should be longer than 4 characters", nil)
		return
	}

	orgId, err := models.CreateOrgV1(models.CreateOrgV1Opts{
		Db:     db,
		Code:   input.Code,
		Name:   input.Name,
		Type:   models.TypeTenantOrg,
		UserId: session.UserId,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create org", err)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("successfully created org[%s] with id[%s]", input.Code, orgId))

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleCreateOrgV1Output{
		Id:   orgId,
		Code: input.Code,
	})
}

// handleGetCurrentOrgV1 godoc
// @Summary      Retrieves the current organisation
// @Description  Retrieves the current organisation that the current user is signed in via
// @Tags         controller-service
// @Accept       json
// @Produce      json
// @Success      200 {object} commonHttpResponse "ok"
// @Failure      403 {object} commonHttpResponse "forbidden"
// @Failure      500 {object} commonHttpResponse "internal server error"
// @Router       /api/v1/org [get]
func handleGetCurrentOrgV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)

	if session.OrganizationId == nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "user is not logged into an organisation", nil)
		return
	}

	log(common.LogLevelDebug, fmt.Sprintf("retrieving org[%s]...", *session.OrganizationId))

	orgInstance, err := models.GetOrgV1(models.GetOrgV1Opts{
		Db: db,
		Id: session.OrganizationId,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to retrieve org[%s]", session.OrganizationId), err)
		return
	}
	if _, err := orgInstance.LoadUserCountV1(models.LoadOrgUserCountV1Opts{Db: db}); err != nil {
		log(common.LogLevelWarn, fmt.Sprintf("failed to get user count of org[%s]: %s", *session.OrganizationId, err))
	}
	log(common.LogLevelDebug, fmt.Sprintf("successfully retrieved org[%s]", *session.OrganizationId))

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", orgInstance)
}
