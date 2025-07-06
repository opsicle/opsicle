package approver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opsicle/internal/common"

	"github.com/gorilla/mux"
)

func getCreateApprovalRequestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ApprovalRequest
		log := r.Context().Value("logger").(requestLogger)

		log(common.LogLevelDebug, "reading request body...")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to read request body: %s", err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to read request body",
				Success: false,
			})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}

		log(common.LogLevelDebug, "parsing request body...")
		err = json.Unmarshal(body, &req.Spec)
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to parse request body: %s", err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to parse request body",
				Success: false,
			})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}
		req.Spec.Init()

		if err := CreateApprovalRequest(req); err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to create approval request: %s", err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to create approval request in persistent data",
				Success: false,
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
			return
		}

		log(common.LogLevelDebug, fmt.Sprintf("sending approvalRequest[%s]...", req.Spec.GetUuid()))
		notificationId, notifications, err := Notifier.SendApprovalRequest(req)
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to send messages for approvalRequest[%v]: %s", req.Spec.Id, err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to send approval request message to telegram",
				Success: false,
			})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(res)
			return
		}
		log(common.LogLevelInfo, fmt.Sprintf("sent %v notifications for approvalRequest[%s]", len(notifications), req.Spec.Id))

		req.Spec.NotificationId = &notificationId
		if err := UpdateApprovalRequest(req); err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to update approval request: %s", err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to update approval request in persistent data",
				Success: false,
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
			return
		}

		res, _ := json.Marshal(httpResponse{
			Data:    req.Spec,
			Message: "ok",
			Success: true,
		})
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	}
}

func getGetApprovalHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := r.Context().Value("logger").(requestLogger)
		approvalId := mux.Vars(r)["approvalId"]
		log(common.LogLevelDebug, fmt.Sprintf("received request for status of approval[%s:%s]", approvalId))

		cacheKey := CreateApprovalCacheKey(approvalId)
		log(common.LogLevelDebug, fmt.Sprintf("retrieving cache item with key[%s]...", cacheKey))
		approvalData, err := Cache.Get(cacheKey)
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to retrieve cache item with key[%s]: %s", cacheKey, err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to retrieve cache item",
				Success: false,
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
			return
		}
		var approval Approval
		if err := json.Unmarshal([]byte(approvalData), &approval); err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to unmarshal approval[%s]: %s", approvalId, err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to unmarshal approval",
				Success: false,
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
			return
		}
		res, _ := json.Marshal(httpResponse{
			Data:    approval.Spec,
			Success: true,
		})
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	}
}

func getGetApprovalRequestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := r.Context().Value("logger").(requestLogger)
		requestId := mux.Vars(r)["requestId"]
		requestUuid := mux.Vars(r)["requestUuid"]
		log(common.LogLevelDebug, fmt.Sprintf("received request for status of approvalRequest[%s:%s]", requestId, requestUuid))

		cacheKey := CreateApprovalRequestCacheKey(requestId, requestUuid)
		log(common.LogLevelDebug, fmt.Sprintf("retrieving cache item with key[%s]...", cacheKey))
		approvalRequestData, err := Cache.Get(cacheKey)
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to retrieve cache item with key[%s]: %s", cacheKey, err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to retrieve cache item",
				Success: false,
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
			return
		}
		var approvalRequest ApprovalRequest
		if err := json.Unmarshal([]byte(approvalRequestData), &approvalRequest); err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to unmarshal approvalRequest[%s]: %s", requestUuid, err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to unmarshal approval request",
				Success: false,
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
			return
		}
		res, _ := json.Marshal(httpResponse{
			Data:    approvalRequest.Spec,
			Success: true,
		})
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	}
}

func getListApprovalRequestsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := r.Context().Value("logger").(requestLogger)
		keys, err := Cache.Scan(CreateApprovalRequestCacheKey("*"))
		if err != nil {
			log(common.LogLevelError, fmt.Sprintf("failed to retrieve approval requests: %s", err))
			res, _ := json.Marshal(httpResponse{
				Message: "failed to retrieve approvals",
				Success: false,
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
			return
		}

		if len(keys) == 0 {
			res, _ := json.Marshal(httpResponse{
				Message: "no approval requests found",
				Success: false,
			})
			w.WriteHeader(http.StatusNotFound)
			w.Write(res)
			return
		}

		res, _ := json.Marshal(httpResponse{
			Data:    keys,
			Success: true,
		})
		w.WriteHeader(http.StatusNotFound)
		w.Write(res)

	}
}
