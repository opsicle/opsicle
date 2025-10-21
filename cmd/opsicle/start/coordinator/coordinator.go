package coordinator

import (
	"context"
	"crypto/tls"
	"fmt"
	"opsicle/internal/audit"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/coordinator"
	"opsicle/internal/database"
	"opsicle/internal/queue"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	ServiceId string = "opsicle/coordinator"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "ca-path",
		DefaultValue: "./certs/ca.crt",
		Usage:        "Specifies the path to a Certificate Authority",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "cert-path",
		DefaultValue: "./certs/server.crt",
		Usage:        "Specifies the path to a server TLS certificate",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "cert-key-path",
		DefaultValue: "./certs/server.key",
		Usage:        "Specifies the path to the key of the TLS certificate as defined in --cert-path",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "controller-url",
		Short:        'u',
		DefaultValue: "http://localhost:54321",
		Usage:        "Defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "grpc-listen-addr",
		DefaultValue: "0.0.0.0:12345",
		Usage:        "Specifies the listen address of the grpc server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "http-listen-addr",
		DefaultValue: "0.0.0.0:12346",
		Usage:        "Specifies the listen address of the http server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-host",
		DefaultValue: "127.0.0.1",
		Usage:        "Specifies the hostname of the MongoDB instance",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mongo-port",
		DefaultValue: "27017",
		Usage:        "Specifies the port which the MongoDB instance is listening on",
		Type:         cli.FlagTypeString,
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
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "coordinator",
	Aliases: []string{"C"},
	Short:   "Starts the coordinator component",
	Long:    "Starts the coordinator component which serves as the API layer that worker interfaces can connect to to receive jobs",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Debugf("starting logging engine...")
		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)
		logrus.Debugf("started logging engine")

		hostname, _ := os.Hostname()
		userId := os.Getuid()
		serviceInstanceId := fmt.Sprintf("%s@%v@%s", ServiceId, userId, hostname)

		certPath := viper.GetString("cert-path")
		if certPath == "" {
			return fmt.Errorf("tls certificate path not provided")
		}
		certKeyPath := viper.GetString("cert-key-path")
		if certKeyPath == "" {
			return fmt.Errorf("tls certificate key path not provided")
		}
		caPath := viper.GetString("ca-path")
		if caPath == "" {
			return fmt.Errorf("ca certificate path not provided")
		}

		serverCert, err := tls.LoadX509KeyPair(certPath, certKeyPath)
		if err != nil {
			return fmt.Errorf("load tls certificate: %w", err)
		}
		caPem, err := os.ReadFile(caPath)
		if err != nil {
			return fmt.Errorf("load ca pem: %w", err)
		}

		/*
		    _  _   _ ___ ___ _____   ___   _ _____ _   ___   _   ___ ___
		   /_\| | | |   |_ _|_   _| |   \ /_|_   _/_\ | _ ) /_\ / __| __|
		  / _ | |_| | |) | |  | |   | |) / _ \| |/ _ \| _ \/ _ \\__ | _|
		 /_/ \_\___/|___|___| |_|   |___/_/ \_|_/_/ \_|___/_/ \_|___|___|

		*/

		logrus.Infof("establishing connection to audit database...")

		auditDatabaseConnection, err := database.ConnectMongo(database.ConnectOpts{
			ConnectionId: ServiceId,
			Host:         viper.GetString("mongo-host"),
			Port:         viper.GetInt("mongo-port"),
			Username:     viper.GetString("mongo-user"),
			Password:     viper.GetString("mongo-password"),
		})
		if err != nil {
			return fmt.Errorf("failed to establish connection to audit database: %w", err)
		}
		logrus.Debugf("established connection to audit database")
		logrus.Infof("starting audit database connection freshness verifier...")
		auditDatabaseConnectionOk := false
		auditDatabaseConnectionStatusLastUpdatedAt := time.Now()
		auditDatabaseConnectionStatusUpdates := make(chan bool)
		var auditDatabaseConnectionStatusMutex sync.Mutex
		var auditModuleError error = nil
		var auditModuleErrorMutex sync.Mutex
		go func() {
			for {
				if auditModuleError == nil {
					logrus.Trace("audit module is ok")
					<-time.After(3 * time.Second)
					continue
				}
				if auditDatabaseConnectionOk {
					logrus.Tracef("(re)trying initialisation of audit module (last error: %s)...", auditModuleError)
					auditModuleErrorMutex.Lock()
					auditModuleError = audit.InitMongo(auditDatabaseConnection)
					if auditModuleError != nil {
						logrus.Errorf("failed to initialise audit module: %s", auditModuleError)
					}
					auditModuleErrorMutex.Unlock()
				} else {
					logrus.Tracef("audit module is not ok (error: %s), waiting for audit database restoration...", auditModuleError)
				}
				<-time.After(3 * time.Second)
			}
		}()
		go func() {
			for {
				statusUpdate := <-auditDatabaseConnectionStatusUpdates
				auditDatabaseConnectionStatusMutex.Lock()
				if statusUpdate != auditDatabaseConnectionOk {
					logAtLevel := logrus.Infof
					if !statusUpdate {
						logAtLevel = logrus.Warnf
						auditModuleError = fmt.Errorf("database connection lost")
					}
					logAtLevel("audit database connection freshness status switched to '%v'", statusUpdate)
					auditDatabaseConnectionStatusLastUpdatedAt = time.Now()
				}
				auditDatabaseConnectionOk = statusUpdate
				auditDatabaseConnectionStatusMutex.Unlock()
			}
		}()
		go func() {
			for {
				logrus.Tracef("verifying audit database connection freshness...")
				if err := database.CheckMongoConnection(ServiceId); err != nil {
					logrus.Errorf("failed to check mongo connection with id '%s': %s", ServiceId, err)
					auditDatabaseConnectionStatusUpdates <- false
					if err := database.RefreshMongoConnection(ServiceId); err != nil {
						logrus.Errorf("failed to refresh mongo connection with id '%s': %s", ServiceId, err)
					} else {
						if err := audit.InitMongo(auditDatabaseConnection); err != nil {
							logrus.Errorf("failed to re-initialise audit module: %s", err)
						}
					}
				} else {
					logrus.Tracef("audit database connection freshness verified")
					auditDatabaseConnectionStatusUpdates <- true
				}
				<-time.After(3 * time.Second)
			}
		}()
		if auditModuleError = audit.InitMongo(auditDatabaseConnection); auditModuleError != nil {
			return fmt.Errorf("failed to initialise audit module: %w", auditModuleError)
		}
		audit.Log(audit.LogEntry{
			EntityId:     fmt.Sprintf("%v@%s", userId, hostname),
			EntityType:   audit.ControllerEntity,
			Verb:         audit.Connect,
			ResourceId:   fmt.Sprintf("%s:%v", viper.GetString("mongo-host"), viper.GetInt("mongo-port")),
			ResourceType: audit.DbResource,
		})

		/*
		   ___  _   _ ___ _   _ ___
		  / _ \| | | | __| | | | __|
		 | (_) | |_| | _|| |_| | _|
		  \__\_\\___/|___|\___/|___|

		*/
		logrus.Infof("establishing connection to queue...")
		nats, err := queue.InitNats(queue.InitNatsOpts{
			Id:          ServiceId,
			Addr:        viper.GetString("nats-addr"),
			Username:    viper.GetString("nats-username"),
			Password:    viper.GetString("nats-password"),
			NKey:        viper.GetString("nats-nkey-value"),
			ServiceLogs: serviceLogs,
		})
		if err != nil {
			return fmt.Errorf("failed to initialise nats queue: %w", err)
		}
		if err := nats.Connect(); err != nil {
			return fmt.Errorf("failed to connect to nats: %w", err)
		}
		logrus.Debugf("established connection to queue")
		audit.Log(audit.LogEntry{
			EntityId:     serviceInstanceId,
			EntityType:   audit.CoordinatorEntity,
			Verb:         audit.Connect,
			ResourceId:   viper.GetString("nats-addr"),
			ResourceType: audit.CacheResource,
		})

		coodinatorContext := context.Background()
		coordinatorInstance, err := coordinator.New(coordinator.NewOpts{
			Context:  coodinatorContext,
			HttpAddr: viper.GetString("http-listen-addr"),
			ReadinessChecks: []func() error{
				func() error {
					if !auditDatabaseConnectionOk {
						return fmt.Errorf("audit database connection is pending restoration")
					}
					return nil
				},
			},
			LivenessChecks: []func() error{
				func() error {
					if !auditDatabaseConnectionOk && auditDatabaseConnectionStatusLastUpdatedAt.Before(time.Now().Add(-30*time.Second)) {
						return fmt.Errorf("audit database connection is invalid")
					}
					return nil
				},
			},
			GrpcAddr: viper.GetString("grpc-listen-addr"),
			GrpcCert: serverCert,
			GrpcCa:   caPem,
			Services: coordinator.Services{
				Queue: nats,
			},
			ServiceLogs: serviceLogs,
		})
		if err != nil {
			return fmt.Errorf("failed to initialise coordinator component")
		}
		if err := coordinatorInstance.Start(); err != nil {
			return fmt.Errorf("failed to start coordinator component")
		}

		return nil
	},
}
