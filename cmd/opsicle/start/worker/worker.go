package worker

import (
	"fmt"
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

const cmdCtx = "o-start-worker-"

func init() {
	currentFlag := "controller-url"
	Command.Flags().StringP(
		currentFlag,
		"u",
		"localhost:12345",
		"the url of the controller",
	)
	viper.BindPFlag(cmdCtx+currentFlag, Command.Flags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "filesystem-path"
	Command.Flags().StringP(
		currentFlag,
		"p",
		"",
		"path to a directory containing automations",
	)
	viper.BindPFlag(cmdCtx+currentFlag, Command.Flags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "runtime"
	Command.Flags().StringP(
		currentFlag,
		"r",
		common.RuntimeDocker,
		fmt.Sprintf("runtime to use, one of ['%s']", strings.Join(common.Runtimes, "', '")),
	)
	viper.BindPFlag(cmdCtx+currentFlag, Command.Flags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "poll-interval"
	Command.Flags().DurationP(
		currentFlag,
		"i",
		time.Second*5,
		"interval between polls",
	)
	viper.BindPFlag(cmdCtx+currentFlag, Command.Flags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)
}

var Command = &cobra.Command{
	Use:   "worker",
	Short: "Starts the worker component",
	Long:  "Starts the worker component that subscribes to the controller and polls for jobs to start",
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString(cmdCtx + "controller-url")
		filesystemPath := viper.GetString(cmdCtx + "filesystem-path")
		pollInterval := viper.GetDuration(cmdCtx + "poll-interval")
		runtime := viper.GetString(cmdCtx + "runtime")

		source := ""
		mode := worker.ModeFilesystem
		if filesystemPath != "" {
			source = filesystemPath
		} else if controllerUrl != "" {
			mode = worker.ModeController
			source = controllerUrl
		}
		if source == "" {
			return fmt.Errorf("failed to identify a worker mode, specify only the controller url or the filesystem path")
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
