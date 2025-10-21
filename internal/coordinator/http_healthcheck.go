package coordinator

import (
	"errors"
	"net/http"
	"opsicle/internal/common"

	"github.com/gorilla/mux"
)

type healthcheckProbes []func() error

type handleHealthcheckProbeOutput struct {
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Status   string   `json:"status"`
}

type RouteRegistrationOpts struct {
	Router      *mux.Router
	ServiceLogs chan<- common.ServiceLog
}

type HealthcheckOpts struct {
	LivenessChecks  healthcheckProbes
	ReadinessChecks healthcheckProbes
}

func registerHealthcheckRoutes(opts RouteRegistrationOpts, hcOpts HealthcheckOpts) {
	opts.Router.HandleFunc("/healthz", getLivenessProbeHandler(hcOpts.LivenessChecks)).Methods(http.MethodGet)
	opts.Router.HandleFunc("/readyz", getReadinessProbeHandler(hcOpts.ReadinessChecks)).Methods(http.MethodGet)
}

func getLivenessProbeHandler(livenessChecks healthcheckProbes) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "大丈夫", handleHealthcheckProbeOutput{
			Errors:   nil,
			Warnings: nil,
			Status:   "ok",
		})
	}
}

func getReadinessProbeHandler(readinessChecks healthcheckProbes) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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
		common.SendHttpSuccessResponse(w, r, http.StatusOK, "大丈夫", handleHealthcheckProbeOutput{
			Errors:   nil,
			Warnings: nil,
			Status:   "ok",
		})
	}
}
