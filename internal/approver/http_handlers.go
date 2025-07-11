package approver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/common"

	"github.com/gorilla/mux"
)

var routesMapping = map[string]map[string]func() http.HandlerFunc{
	"/approval-request": {
		http.MethodGet:  getListApprovalRequestsHandler,
		http.MethodPost: getCreateApprovalRequestHandler,
	},
	"/approval/{approvalId}": {
		http.MethodGet: getGetApprovalHandler,
	},
	"/approval-request/{requestId}/{requestUuid}": {
		http.MethodGet: getGetApprovalRequestHandler,
	},
}

func getCreateApprovalRequestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ApprovalRequest
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
		if err := CreateApprovalRequest(req); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to create approval request", err)
			return
		}

		log(common.LogLevelDebug, fmt.Sprintf("sending approvalRequest[%s]...", req.Spec.GetUuid()))
		requestUuid, notifications, err := Notifiers.SendApprovalRequest(&req)
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to send approvalRequest[%v:%s]", req.Spec.Id, requestUuid), err)
			return
		}
		log(common.LogLevelInfo, fmt.Sprintf("sent %v notifications for approvalRequest[%s:%s]", len(notifications), req.Spec.Id, requestUuid))

		if err := UpdateApprovalRequest(req); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to update approvalRequest[%v:%s]", req.Spec.Id, requestUuid), err)
			return
		}
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", req.Spec)
	}
}

func getGetApprovalHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
		approvalId := mux.Vars(r)["approvalId"]
		log(common.LogLevelDebug, fmt.Sprintf("received request for status of approval[%s:%s]", approvalId))

		cacheKey := CreateApprovalCacheKey(approvalId)
		log(common.LogLevelDebug, fmt.Sprintf("retrieving cache item with key[%s]...", cacheKey))
		approvalData, err := Cache.Get(cacheKey)
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to retrieve cache item with key[%s]", cacheKey), err)
			return
		}
		var approval Approval
		if err := json.Unmarshal([]byte(approvalData), &approval); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal approval[%s]", approvalId), err)
			return
		}
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", approval.Spec)
	}
}

func getGetApprovalRequestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
		requestId := mux.Vars(r)["requestId"]
		requestUuid := mux.Vars(r)["requestUuid"]
		log(common.LogLevelDebug, fmt.Sprintf("received request for status of approvalRequest[%s:%s]", requestId, requestUuid))

		cacheKey := CreateApprovalRequestCacheKey(requestId, requestUuid)
		log(common.LogLevelDebug, fmt.Sprintf("retrieving cache item with key[%s]...", cacheKey))
		approvalRequestData, err := Cache.Get(cacheKey)
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to retrieve cache item with key[%s]", cacheKey), err)
			return
		}
		var approvalRequest ApprovalRequest
		if err := json.Unmarshal([]byte(approvalRequestData), &approvalRequest); err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal approvalRequest[%s]", requestUuid), err)
			return
		}
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", approvalRequest.Spec)
	}
}

func getListApprovalRequestsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)

		log(common.LogLevelDebug, "retrieving all cache keys...")
		keys, err := Cache.Scan(CreateApprovalRequestCacheKey("*"))
		if err != nil {
			common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "failed to retrieve approval requests", err)
			return
		}

		if len(keys) == 0 {
			common.SendHttpSuccessResponse(w, r, http.StatusNotFound, "no approval requests found")
			return
		}

		common.SendHttpSuccessResponse(w, r, http.StatusOK, "ok", keys)
	}
}
