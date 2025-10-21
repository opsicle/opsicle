package worker

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/worker"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "coordinator-url",
		Short:        'u',
		DefaultValue: "localhost:12345",
		Usage:        "the url of the coordinator",
		Type:         cli.FlagTypeString,
	},

	{
		Name:         "filesystem-path",
		Short:        'p',
		DefaultValue: "",
		Usage:        "path to a directory containing automations",
		Type:         cli.FlagTypeString,
	},

	{
		Name:         "runtime",
		Short:        'r',
		DefaultValue: common.RuntimeDocker,
		Usage:        fmt.Sprintf("runtime to use, one of ['%s']", strings.Join(common.Runtimes, "', '")),
		Type:         cli.FlagTypeString,
	},

	{
		Name:         "poll-interval",
		Short:        'i',
		DefaultValue: time.Second * 5,
		Usage:        "interval between polls",
		Type:         cli.FlagTypeDuration,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "worker",
	Short: "Starts the worker component",
	Long:  "Starts the worker component that subscribes to the coordinator and polls for jobs to start",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		coordinatorUrl := viper.GetString("coordinator-url")
		filesystemPath := viper.GetString("filesystem-path")
		pollInterval := viper.GetDuration("poll-interval")
		runtime := viper.GetString("runtime")

		source := ""
		mode := worker.ModeFilesystem
		if filesystemPath != "" {
			source = filesystemPath
		} else if coordinatorUrl != "" {
			mode = worker.ModeCoordinator
			source = coordinatorUrl
		}
		if source == "" {
			return fmt.Errorf("failed to identify a worker mode, specify only the coordinator url or the filesystem path")
		}
		serviceLogs := make(chan common.ServiceLog, 64)
		go func() {
			for {
				serviceLog, ok := <-serviceLogs
				if !ok {
					return
				}
				log := logrus.Info
				switch serviceLog.Level {
				case common.LogLevelTrace:
					log = logrus.Trace
				case common.LogLevelDebug:
					log = logrus.Debug
				case common.LogLevelInfo:
					log = logrus.Info
				case common.LogLevelWarn:
					log = logrus.Warn
				case common.LogLevelError:
					log = logrus.Error
				}
				log(serviceLog.Message)
			}
		}()
		automationLogs := make(chan string, 64)
		go func() {
			for {
				automationLog, ok := <-automationLogs
				if !ok {
					break
				}
				fmt.Print(automationLog)
			}
		}()
		doneChannel := make(chan common.Done)
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		go func() {
			sig := <-sigs
			logrus.Infof("received signal: %s", sig)
			doneChannel <- common.Done{}
		}()
		workerInstance := worker.NewWorker(worker.NewWorkerOpts{
			AutomationLogs: &automationLogs,
			DoneChannel:    doneChannel,
			ServiceLogs:    &serviceLogs,
			Mode:           mode,
			PollInterval:   pollInterval,
			Runtime:        runtime,
			Source:         source,
		})
		if err := workerInstance.Start(); err != nil {
			logrus.Errorf("failed to start worker instance: %s", err)
			os.Exit(1)
		}
		logrus.Infof("ok: exitting with status code 0")
		return nil
	},
}
