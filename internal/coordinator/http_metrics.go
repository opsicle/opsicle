package coordinator

import "github.com/prometheus/client_golang/prometheus/promhttp"

func registerMetricsRoutes(opts RouteRegistrationOpts) {
	opts.Router.Handle("/metrics", promhttp.Handler())
}
