package common

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type CommonHttpEndpointsOpts struct {
	Router          *mux.Router
	ServiceLogs     chan<- ServiceLog
	LivenessChecks  []func() error
	ReadinessChecks []func() error
}

func RegisterCommonHttpEndpoints(opts CommonHttpEndpointsOpts) {
	opts.Router.HandleFunc("/healthz", getLivenessProbeHandler(opts)).Methods(http.MethodGet)
	opts.Router.HandleFunc("/readyz", getReadinessProbeHandler(opts)).Methods(http.MethodGet)
	opts.Router.Handle("/metrics", promhttp.Handler())
}

type handleHealthcheckProbeOutput struct {
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Status   string   `json:"status"`
}

func getLivenessProbeHandler(opts CommonHttpEndpointsOpts) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		isConsideredLive := true
		livenessIssues := []error{}
		for _, livenessCheck := range opts.LivenessChecks {
			if err := livenessCheck(); err != nil {
				isConsideredLive = false
				livenessIssues = append(livenessIssues, err)
			}
		}
		if !isConsideredLive {
			SendHttpFailResponse(w, r, http.StatusInternalServerError, "大丈夫じゃない", errors.Join(livenessIssues...))
			return
		}
		SendHttpSuccessResponse(w, r, http.StatusOK, "大丈夫", handleHealthcheckProbeOutput{
			Errors:   nil,
			Warnings: nil,
			Status:   "ok",
		})
	}
}

func getReadinessProbeHandler(opts CommonHttpEndpointsOpts) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		isConsideredReady := true
		readinessIssues := []error{}
		for _, readinessCheck := range opts.ReadinessChecks {
			if err := readinessCheck(); err != nil {
				isConsideredReady = false
				readinessIssues = append(readinessIssues, err)
			}
		}
		if !isConsideredReady {
			SendHttpFailResponse(w, r, http.StatusInternalServerError, "大丈夫じゃない", errors.Join(readinessIssues...))
			return
		}
		SendHttpSuccessResponse(w, r, http.StatusOK, "大丈夫", handleHealthcheckProbeOutput{
			Errors:   nil,
			Warnings: nil,
			Status:   "ok",
		})
	}
}
