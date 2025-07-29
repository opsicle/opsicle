package controller

import (
	"errors"
	"net/http"
	"opsicle/internal/common"
)

var (
	livenessChecks  []func() error
	readinessChecks []func() error
)

func registerHealthcheckRoutes(opts RouteRegistrationOpts) {
	opts.Router.HandleFunc("/healthz", handleLivenessProbe).Methods(http.MethodGet)
	opts.Router.HandleFunc("/readyz", handleReadinessProbe).Methods(http.MethodGet)
}

func handleLivenessProbe(w http.ResponseWriter, r *http.Request) {
	isConsideredLive := true
	livenessIssues := []error{}
	for _, livenessCheck := range livenessChecks {
		if err := livenessCheck(); err != nil {
			isConsideredLive = false
			livenessIssues = append(livenessIssues, err)
		}
	}
	if !isConsideredLive {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "大丈夫じゃない", errors.Join(livenessIssues...))
		return
	}
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "大丈夫", nil)
}

func handleReadinessProbe(w http.ResponseWriter, r *http.Request) {
	isConsideredReady := true
	readinessIssues := []error{}
	for _, readinessCheck := range readinessChecks {
		if err := readinessCheck(); err != nil {
			isConsideredReady = false
			readinessIssues = append(readinessIssues, err)
		}
	}
	if !isConsideredReady {
		common.SendHttpFailResponse(w, r, http.StatusInternalServerError, "大丈夫じゃない", errors.Join(readinessIssues...))
		return
	}
	common.SendHttpSuccessResponse(w, r, http.StatusOK, "大丈夫", nil)
}
