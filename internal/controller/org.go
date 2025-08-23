package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"
	"opsicle/internal/validate"
	"time"
)

func registerOrgRoutes(opts RouteRegistrationOpts) {
	requiresAuth := getRouteAuther(opts.ServiceLogs)

	v1 := opts.Router.PathPrefix("/v1/org").Subrouter()

	v1.Handle("", requiresAuth(http.HandlerFunc(handleCreateOrgV1))).Methods(http.MethodPost)

	v1 = opts.Router.PathPrefix("/v1/orgs").Subrouter()

	v1.Handle("", requiresAuth(http.HandlerFunc(handleListOrgsV1))).Methods(http.MethodGet)
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
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", ErrorInvalidInput)
		return
	}
	log(common.LogLevelDebug, "successfully read body into bytes")
	var input handleCreateOrgV1Input
	if err := json.Unmarshal(requestBody, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", ErrorInvalidInput)
		return
	}

	log(common.LogLevelDebug, "successfully parsed body into expected input class")

	if err := validate.OrgName(input.Name); err != nil {
		log(common.LogLevelDebug, fmt.Sprintf("user[%s] entered an invalid orgName[%s]: %s", input.Name, session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "org name is invalid", ErrorInvalidInput, err.Error())
		return
	}
	if err := validate.OrgCode(input.Code); err != nil {
		log(common.LogLevelDebug, fmt.Sprintf("user[%s] entered an invalid orgCode[%s]: %s", input.Code, session.UserId, err))
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "org code is invalid", ErrorInvalidInput, err.Error())
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
		if errors.Is(err, models.ErrorDuplicateEntry) {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "org already exists", ErrorOrgExists)
			return
		}
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create org", ErrorDatabaseIssue)
		return
	}
	log(common.LogLevelDebug, fmt.Sprintf("successfully created org[%s] with id[%s]", input.Code, orgId))

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", handleCreateOrgV1Output{
		Id:   orgId,
		Code: input.Code,
	})
}

type handleListOrgsV1Output handleListOrgsV1OutputOrgs

type handleListOrgsV1OutputOrgs []handleListOrgsV1OutputOrg
type handleListOrgsV1OutputOrg struct {
	Code       string     `json:"code"`
	CreatedAt  time.Time  `json:"createdAt"`
	Id         string     `json:"id"`
	JoinedAt   time.Time  `json:"joinedAt"`
	MemberType string     `json:"memberType"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	UpdatedAt  *time.Time `json:"updatedAt"`
}

// handleListOrgsV1 godoc
// @Summary      Retrieves the current organisation
// @Description  Retrieves the current organisation that the current user is signed in via
// @Tags         controller-service
// @Accept       json
// @Produce      json
// @Success      200 {object} commonHttpResponse "ok"
// @Failure      403 {object} commonHttpResponse "forbidden"
// @Failure      500 {object} commonHttpResponse "internal server error"
// @Router       /api/v1/org [get]
func handleListOrgsV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	session := r.Context().Value(authRequestContext).(identity)

	log(common.LogLevelDebug, fmt.Sprintf("retrieving organisations that user[%s] is in", session.UserId))

	orgs, err := models.ListUserOrgsV1(models.ListUserOrgsV1Opts{
		Db:     db,
		UserId: session.UserId,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to retrieve orgs that user[%s] is in", session.UserId), err)
		return
	}

	output := handleListOrgsV1Output{}
	for _, org := range orgs {
		output = append(
			output,
			handleListOrgsV1OutputOrg{
				Code:       org.Code,
				CreatedAt:  org.CreatedAt,
				Id:         *org.Id,
				JoinedAt:   *org.JoinedAt,
				MemberType: *org.MemberType,
				Name:       org.Name,
				Type:       org.Type,
				UpdatedAt:  org.UpdatedAt,
			},
		)
	}

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", output)
}
