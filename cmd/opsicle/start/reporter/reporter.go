package reporter

import (
	"fmt"
	"net/http"
	"opsicle/internal/cli"
	"opsicle/internal/common"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Command = cli.NewCommand(cli.CommandOpts{
	Name:  "reporter",
	Flags: flags,
	Use:   "reporter",
	Short: "Starts the reporter component",
	Long:  "Starts the reporter component which reports on the services",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		appName := opts.GetFullname()
		serviceLogs := opts.GetServiceLogs()

		healthcheckInterval := viper.GetDuration("healthcheck-interval")
		healthchecks := []healthcheck{
			{
				run: startApproverReporter,
				opts: &healthcheckOpts{
					interval: healthcheckInterval,
					stopper:  make(chan common.Done),
					status: prometheus.NewGauge(prometheus.GaugeOpts{
						Name:      "approver_service_up",
						Namespace: appName,
					}),
				},
			},
			{
				run: startMongoReporter,
				opts: &healthcheckOpts{
					interval: healthcheckInterval,
					stopper:  make(chan common.Done),
					status: prometheus.NewGauge(prometheus.GaugeOpts{
						Name:      "mongo_up",
						Namespace: appName,
					}),
				},
			},
			{
				run: startMysqlReporter,
				opts: &healthcheckOpts{
					interval: healthcheckInterval,
					stopper:  make(chan common.Done),
					status: prometheus.NewGauge(prometheus.GaugeOpts{
						Name:      "mysql_up",
						Namespace: appName,
					}),
				},
			},
			{
				run: startNatsReporter,
				opts: &healthcheckOpts{
					interval: healthcheckInterval,
					stopper:  make(chan common.Done),
					status: prometheus.NewGauge(prometheus.GaugeOpts{
						Name:      "nats_up",
						Namespace: appName,
					}),
				},
			},
			{
				run: startRedisReporter,
				opts: &healthcheckOpts{
					interval: healthcheckInterval,
					stopper:  make(chan common.Done),
					status: prometheus.NewGauge(prometheus.GaugeOpts{
						Name:      "redis_up",
						Namespace: appName,
					}),
				},
			},
		}
		for _, healthcheckInstance := range healthchecks {
			prometheus.MustRegister(healthcheckInstance.opts.status)
			healthcheckInstance.run(appName, healthcheckInstance.opts)
		}

		handler := mux.NewRouter()
		handler.Handle("/metrics", promhttp.Handler())
		handler.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		}))
		handler.Handle("/readyz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		}))

		serverStopper := make(chan common.Done)
		server, err := common.NewHttpServer(common.NewHttpServerOpts{
			Addr:        viper.GetString("listen-addr"),
			Done:        serverStopper,
			Handler:     handler,
			ServiceLogs: serviceLogs,
		})
		if err != nil {
			return fmt.Errorf("failed to create http server: %w", err)
		}
		if err := server.Start(); err != nil {
			return fmt.Errorf("failed to start http server: %w", err)
		}

		return nil
	},
})
