package automation

import (
	"fmt"
	"opsicle/internal/automations"
	"opsicle/internal/common"
	"opsicle/internal/config"
	"opsicle/internal/worker"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	currentFlag := "automation-path"
	Command.PersistentFlags().StringP(
		currentFlag,
		"p",
		"",
		"path to the automation manifest",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
}

var Command = &cobra.Command{
	Use:     "automation <path-to-automation>",
	Aliases: []string{"a"},
	Short:   "Runs an Automation resource independently",
	RunE: func(cmd *cobra.Command, args []string) error {
		resourceIsSpecified := false
		resourcePath := ""
		if len(args) > 0 {
			resourcePath = args[0]
			resourceIsSpecified = true
		}
		if !resourceIsSpecified {
			return fmt.Errorf("failed to receive a <path-to-template-file")
		}
		fi, err := os.Stat(resourcePath)
		if err != nil {
			return fmt.Errorf("failed to check for existence of file at path[%s]: %s", resourcePath, err)
		}
		if fi.IsDir() {
			return fmt.Errorf("failed to get a file at path[%s]: got a directory", resourcePath)
		}
		automationInstance, err := automations.LoadFromFile(resourcePath)
		if err != nil {
			return fmt.Errorf("failed to load automation from path[%s]: %s", resourcePath, err)
		}
		var logsWaiter sync.WaitGroup
		workerLogs := make(chan worker.LogEntry, 64)
		automationLogs := make(chan string, 64)
		doneEventChannel := make(chan common.Done)
		logsWaiter.Add(1)
		go func() {
			<-doneEventChannel
			close(workerLogs)
		}()
		logsWaiter.Add(1)
		go func() {
			// wait for the logs to finish, otherwise some logs
			// might not be printed
			defer logsWaiter.Done()
			for {
				automationLog, ok := <-automationLogs
				if !ok {
					break
				}
				fmt.Print(automationLog)
			}
		}()
		go func() {
			defer close(automationLogs)
			logrus.Infof("started worker logging event loop for automation[%s]", automationInstance.Resource.Metadata.Name)
			for {
				workerLog, ok := <-workerLogs
				if !ok {
					break
				}
				logger := logrus.Info
				switch workerLog.Level {
				case config.LogLevelTrace:
					logger = logrus.Trace
				case config.LogLevelDebug:
					logger = logrus.Debug
				case config.LogLevelInfo:
					logger = logrus.Info
				case config.LogLevelWarn:
					logger = logrus.Warn
				case config.LogLevelError:
					logger = logrus.Error
				}
				logger(workerLog.Message)
			}
			logrus.Infof("worker logs have stopped streaming for automation[%s]", automationInstance.Resource.Metadata.Name)
			logsWaiter.Done()
		}()
		logsWaiter.Add(1)
		if err := worker.RunAutomation(worker.RunAutomationOpts{
			Done:           &doneEventChannel,
			Spec:           automationInstance,
			ServiceLogs:    workerLogs,
			AutomationLogs: automationLogs,
		}); err != nil {
			return fmt.Errorf("automation execution failed with message: %s", err)
		}
		logsWaiter.Done()
		logsWaiter.Wait()
		return nil
	},
}
