package approver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/cache"
	"opsicle/internal/common"

	"opsicle/pkg/approver"

	"github.com/gorilla/mux"
)

var routesMapping = map[string]map[string]func() http.HandlerFunc{
	"/api/v1/approval-request": {
		http.MethodGet:  getListApprovalRequestsHandler,
		http.MethodPost: getCreateApprovalRequestHandler,
	},
	"/api/v1/approval/{approvalUuid}": {
		http.MethodGet: getGetApprovalHandler,
	},
	"/api/v1/approval-request/{requestUuid}": {
		http.MethodGet: getGetApprovalRequestHandler,
	},
}

type commonHttpResponse common.HttpResponse
type createApprovalRequestInput approver.CreateApprovalRequestInput

// getCreateApprovalRequestHandler godoc
// @Summary      Creates approval requests
// @Description  This endpoint creates approval requests
// @Tags         approver-service
// @Accept       json
// @Produce      json
// @Security     BasicAuth
// @Param        request body approver.CreateApprovalRequestInput true "Approval payload"
// @Success      200 {object} commonHttpResponse "approved"
// @Failure      400 {object} commonHttpResponse "bad request"
// @Failure      500 {object} commonHttpResponse "internal server error" {"success": false}
// @Router       /api/v1/approval-request [post]
func getCreateApprovalRequestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &ApprovalRequest{}
		log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)

		log(common.LogLevelDebug, "reading request body...")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to read request body", err)
			return
		}

		log(common.LogLevelDebug, "parsing request body...")
		err = json.Unmarshal(body, &req.Spec)
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusBadRequest, "failed to parse request body", err)
			return
		}

		log(common.LogLevelDebug, "storing approval request...")
		req.Spec.Init()
		if err := req.Create(); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create approval request", err)
			return
		}

		log(common.LogLevelDebug, fmt.Sprintf("sending approvalRequest[%s]...", req.Spec.GetUuid()))
		requestUuid, notifications, err := Notifiers.SendApprovalRequest(req)
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to send approvalRequest[%v:%s]", req.Spec.Id, requestUuid), err)
			return
		}
		log(common.LogLevelInfo, fmt.Sprintf("sent %v notifications for approvalRequest[%s:%s]", len(notifications), req.Spec.Id, requestUuid))

		if err := req.Update(); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to update approvalRequest[%v:%s]", req.Spec.Id, requestUuid), err)
			return
		}
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", req.Spec)
	}
}

// getGetApprovalHandler godoc
// @Summary      Retreives an approval given it's ID
// @Description  This endpoint retrieves an approval given it's ID
// @Tags         approver-service
// @Accept       json
// @Produce      json
// @Security     BasicAuth
// @Param				 approvalUuid path string true "Approval UUID"
// @Success      200 {object} commonHttpResponse "Success"
// @Failure      404 {object} commonHttpResponse "Not found"
// @Failure      500 {object} commonHttpResponse "Internal server error"
// @Router       /api/v1/approval/{approvalUuid} [get]
func getGetApprovalHandler() http.HandlerFunc {
	cacheInstance := cache.Get()
	return func(w http.ResponseWriter, r *http.Request) {
		log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
		approvalUuid := mux.Vars(r)["approvalUuid"]
		log(common.LogLevelDebug, fmt.Sprintf("received request for status of approval[%s]", approvalUuid))

		cacheKey := CreateApprovalCacheKey(approvalUuid)
		log(common.LogLevelDebug, fmt.Sprintf("retrieving cache item with key[%s]...", cacheKey))
		approvalData, err := cacheInstance.Get(cacheKey)
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to retrieve cache item with key[%s]", cacheKey), err)
			return
		}
		var approval Approval
		if err := json.Unmarshal([]byte(approvalData), &approval); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal approval[%s]", approvalUuid), err)
			return
		}
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", approval.Spec)
	}
}

// getGetApprovalRequestHandler godoc
// @Summary      Retreives all approval requests
// @Description  This endpoint retrieves all approval requests
// @Tags         approver-service
// @Accept       json
// @Produce      json
// @Security     BasicAuth
// @Param				 requestUuid path string true "Request UUID"
// @Success      200 {object} commonHttpResponse "Success"
// @Failure      404 {object} commonHttpResponse "Not found"
// @Failure      500 {object} commonHttpResponse "Internal server error"
// @Router       /api/v1/approval-request/{requestUuid} [get]
func getGetApprovalRequestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cacheInstance := cache.Get()
		log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
		requestUuid := mux.Vars(r)["requestUuid"]
		log(common.LogLevelDebug, fmt.Sprintf("received request for status of approvalRequest[%s]", requestUuid))

		cacheKey := CreateApprovalRequestCacheKey(requestUuid)
		log(common.LogLevelDebug, fmt.Sprintf("retrieving cache item with key[%s]...", cacheKey))
		approvalRequestData, err := cacheInstance.Get(cacheKey)
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to retrieve cache item with key[%s]", cacheKey), err)
			return
		}
		var approvalRequest ApprovalRequest
		if err := json.Unmarshal([]byte(approvalRequestData), &approvalRequest); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal approvalRequest[%s]", requestUuid), err)
			return
		}
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", approvalRequest.GetRedacted())
	}
}

// getListApprovalRequestsHandler godoc
// @Summary      Retreives all approval requests
// @Description  This endpoint retrieves all approval requests
// @Tags         approver-service
// @Accept       json
// @Produce      json
// @Security     BasicAuth
// @Success      200 {object} commonHttpResponse "Success"
// @Failure      404 {object} commonHttpResponse "Not found"
// @Failure      500 {object} commonHttpResponse "Internal server error"
// @Router       /api/v1/approval-request [get]
func getListApprovalRequestsHandler() http.HandlerFunc {
	cacheInstance := cache.Get()
	return func(w http.ResponseWriter, r *http.Request) {
		log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)

		log(common.LogLevelDebug, "retrieving all cache keys...")
		keys, err := cacheInstance.Scan(CreateApprovalRequestCacheKey("*"))
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve approval requests", err)
			return
		}

		if len(keys) == 0 {
			common.SendHttpSuccessResponse(w, r, http.StatusNotFound, "no approval requests found")
			return
		}
		for i := 0; i < len(keys); i++ {
			keys[i] = StripCacheKeyPrefix(keys[i])
		}

		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", keys)
	}
}
