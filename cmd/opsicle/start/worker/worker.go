package worker

import (
	"fmt"
	"opsicle/internal/common"
	"opsicle/internal/config"
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

func init() {
	currentFlag := "controller-url"
	Command.Flags().StringP(
		currentFlag,
		"u",
		"localhost:12345",
		"the url of the controller",
	)
	viper.BindPFlag(currentFlag, Command.Flags().Lookup(currentFlag))

	currentFlag = "filesystem-path"
	Command.Flags().StringP(
		currentFlag,
		"p",
		"",
		"path to a directory containing automations",
	)
	viper.BindPFlag(currentFlag, Command.Flags().Lookup(currentFlag))

	currentFlag = "runtime"
	Command.Flags().StringP(
		currentFlag,
		"r",
		config.RuntimeDocker,
		fmt.Sprintf("runtime to use, one of ['%s']", strings.Join(config.Runtimes, "', '")),
	)

	currentFlag = "poll-interval"
	Command.Flags().DurationP(
		currentFlag,
		"i",
		time.Second*5,
		"interval between polls",
	)
	viper.BindPFlag(currentFlag, Command.Flags().Lookup(currentFlag))
}

var Command = &cobra.Command{
	Use:   "worker",
	Short: "Starts the worker component",
	Long:  "Starts the worker component that subscribes to the controller and polls for jobs to start",
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		filesystemPath := viper.GetString("filesystem-path")
		pollInterval := viper.GetDuration("poll-interval")
		runtime := viper.GetString("runtime")

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
		logChannel := make(chan worker.LogEntry, 64)
		go func() {
			for {
				logEntry, ok := <-logChannel
				if !ok {
					return
				}
				log := logrus.Info
				switch logEntry.Level {
				case config.LogLevelTrace:
					log = logrus.Trace
				case config.LogLevelDebug:
					log = logrus.Debug
				case config.LogLevelInfo:
					log = logrus.Info
				case config.LogLevelWarn:
					log = logrus.Warn
				case config.LogLevelError:
					log = logrus.Error
				}
				log(logEntry.Message)
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
			ServiceLogs:    &logChannel,
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
