package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/auth"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"
	"strings"
)

func registerAdminRoutes(opts RouteRegistrationOpts, adminToken string) {
	requiresAuth := getAdminRouteAuther(adminToken, opts.ServiceLogs)
	v1 := opts.Router.PathPrefix("/v1").Subrouter()
	v1.Use(requiresAuth)

	v1.HandleFunc("/init", initHandlerV1).Methods(http.MethodPost)
}

type initHandlerV1Input struct {
	// Email is the root user's email address
	Email string `json:"email"`
	// Password is the root user's password
	Password string `json:"password"`
}

type initHandlerV1Output struct {
	UserEmail string `json:"userEmail"`
	UserId    string `json:"userId"`
	OrgId     string `json:"orgId"`
	OrgCode   string `json:"orgCode"`
}

func initHandlerV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelInfo, "this endpoint initialises the server")
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body")
		return
	}
	var input initHandlerV1Input
	if err := json.Unmarshal(requestBody, &input); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body")
		return
	}

	log(common.LogLevelDebug, "received request to create a superadmin")

	if !auth.IsEmailValid(input.Email) {
		common.SendHttpFailResponse(w, r, http.StatusBadRequest, "invalid email address", err)
		return
	}

	if err := models.CreateUserV1(models.CreateUserV1Opts{
		Db: db,

		Email:    input.Email,
		Password: input.Password,
		Type:     models.TypeSystemAdmin,
	}); err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create admin user", err)
		return
	}
	adminUser, err := models.GetUserV1(models.GetUserV1Opts{
		Db:    db,
		Email: &input.Email,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve admin user", err)
		return
	}
	rootOrgCode := "root"
	rootOrg, err := models.GetOrgV1(models.GetOrgV1Opts{
		Db:   db,
		Code: &rootOrgCode,
	})
	if err != nil {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to detect root org", err)
		return
	}
	if rootOrg == nil {
		_, err := models.CreateOrgV1(models.CreateOrgV1Opts{
			Db:     db,
			Name:   "Root Organisation",
			Code:   rootOrgCode,
			Type:   models.TypeAdminOrg,
			UserId: *adminUser.Id,
		})
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create root org", err)
			return
		}
		rootOrg, err = models.GetOrgV1(models.GetOrgV1Opts{
			Db:   db,
			Code: &rootOrgCode,
		})
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve created root org", err)
			return
		}
	}

	common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", initHandlerV1Output{
		UserId:    *adminUser.Id,
		UserEmail: adminUser.Email,
		OrgId:     *rootOrg.Id,
		OrgCode:   rootOrg.Code,
	})
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
