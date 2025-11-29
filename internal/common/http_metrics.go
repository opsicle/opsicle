package common

import (
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	prometheus.MustRegister(incomingRequestsCounter)
	prometheus.MustRegister(pendingRequestsCounter)
}

var incomingRequestsCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	},
	[]string{"method", "path"},
)

var pendingRequestsCounter = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "http_requests_pending",
		Help: "Total number of HTTP requests being processed",
	},
	[]string{"method", "path"},
)
