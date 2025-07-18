package automation

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/automations"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/worker"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:     "automation <path-to-automation>",
	Aliases: []string{"a"},
	Short:   "Runs an Automation resource independently",
	RunE: func(cmd *cobra.Command, args []string) error {
		resourcePath, err := cli.GetFilePathFromArgs(args)
		if err != nil {
			return fmt.Errorf("failed to receive required <path-to-automation>: %s", err)
		}
		automationInstance, err := automations.LoadAutomationFromFile(resourcePath)
		if err != nil {
			return fmt.Errorf("failed to load automation from path[%s]: %s", resourcePath, err)
		}
		o, _ := json.MarshalIndent(automationInstance, "", "  ")
		logrus.Debugf("loaded automation as follows:\n%s", string(o))

		var logsWaiter sync.WaitGroup
		serviceLogs := make(chan common.ServiceLog, 64)
		automationLogs := make(chan string, 64)
		doneEventChannel := make(chan common.Done)
		logsWaiter.Add(1)
		go func() {
			<-doneEventChannel
			close(serviceLogs)
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
				workerLog, ok := <-serviceLogs
				if !ok {
					break
				}
				logger := logrus.Info
				switch workerLog.Level {
				case common.LogLevelTrace:
					logger = logrus.Trace
				case common.LogLevelDebug:
					logger = logrus.Debug
				case common.LogLevelInfo:
					logger = logrus.Info
				case common.LogLevelWarn:
					logger = logrus.Warn
				case common.LogLevelError:
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
			ServiceLogs:    serviceLogs,
			AutomationLogs: automationLogs,
		}); err != nil {
			return fmt.Errorf("automation execution failed with message: %s", err)
		}
		logsWaiter.Done()
		logsWaiter.Wait()
		return nil
	},
}
