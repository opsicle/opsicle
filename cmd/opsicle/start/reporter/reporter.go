package reporter

import (
	"fmt"
	"net/http"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "healthcheck-interval",
		DefaultValue: 3 * time.Second,
		Usage:        "specifies the interval between healthcheck pings",
		Type:         cli.FlagTypeDuration,
	},
	{
		Name:         "listen-addr",
		DefaultValue: "0.0.0.0:11111",
		Usage:        "specifies the listen address of the server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-host",
		DefaultValue: []string{"127.0.0.1:27017"},
		Usage:        "Specifies the hostname(s) of the MongoDB instance",
		Type:         cli.FlagTypeStringSlice,
	},
	{
		Name:         "mongo-user",
		DefaultValue: "opsicle",
		Usage:        "Specifies the username to use to login to the MongoDB instance",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-password",
		DefaultValue: "password",
		Usage:        "Specifies the password to use to login to the MongoDB instance",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-host",
		DefaultValue: "127.0.0.1",
		Usage:        "specifies the hostname of the database",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-port",
		DefaultValue: "3306",
		Usage:        "specifies the port which the database is listening on",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-database",
		DefaultValue: "opsicle",
		Usage:        "specifies the name of the central database schema",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-user",
		DefaultValue: "opsicle",
		Usage:        "specifies the username to use to login",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-password",
		DefaultValue: "password",
		Usage:        "specifies the password to use to login",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "nats-addr",
		DefaultValue: "localhost:4222",
		Usage:        "Specifies the hostname (including port) of the NATS server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "nats-username",
		DefaultValue: "opsicle",
		Usage:        "Specifies the username used to login to NATS",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "nats-password",
		DefaultValue: "password",
		Usage:        "Specifies the password used to login to NATS",
		Type:         cli.FlagTypeString,
	},
	{
		Name: "nats-nkey-value",
		// this default value is the development nkey, this value must be aligned
		// to the one in `./docker-compose.yml` in the root of the repository
		DefaultValue: "SUADZTA4VJHBCO7K75DQ3IN7KZGWHKEI26D2IYEABRN5TXXYHXLWNDYT4A",
		Usage:        "Specifies the nkey used to login to NATS",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "redis-addr",
		DefaultValue: "localhost:6379",
		Usage:        "defines the hostname (including port) of the redis server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "redis-user",
		DefaultValue: "opsicle",
		Usage:        "defines the username used to login to redis",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "redis-password",
		DefaultValue: "password",
		Usage:        "defines the password used to login to redis",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "reporter",
	Short: "Starts the reporter component",
	Long:  "Starts the reporter component which reports on the services",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Debugf("starting logging engine...")
		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)
		logrus.Infof("started logging engine")

		appName := "opsicle/reporter"
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
}
