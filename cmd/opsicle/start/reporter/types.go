package reporter

import (
	"opsicle/internal/common"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type healthcheck struct {
	run  func(id string, opts *healthcheckOpts) error
	opts *healthcheckOpts
}

type healthcheckOpts struct {
	status   prometheus.Gauge
	stopper  chan common.Done
	interval time.Duration
}
