package automation

import (
	"encoding/json"
	"errors"
	"fmt"
	"opsicle/internal/automations"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/worker"
	"opsicle/pkg/controller"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "controller-url",
		DefaultValue: "http://localhost:54321",
		Usage:        "Defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "file",
		Short:        'f',
		DefaultValue: "",
		Usage:        "Specifies a filesystem path to the automation; if specified, automation will execute locally",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "template-id",
		Short:        't',
		DefaultValue: "",
		Usage:        "ID (or name) of the template to remove a user from",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "automation",
	Aliases: []string{"a"},
	Short:   "Runs an Automation resource, if --file is specified, the automation will be executed locally",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		automationPath := viper.GetString("file")
		if automationPath != "" {
			return handleLocalExecution(cmd, automationPath)
		}
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/join/org"
		sessionToken, err := cli.RequireAuth(controllerUrl, methodId)
		if err != nil {
			fmt.Println("⚠️  You must be logged-in to run this command")
			return err
		}

		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			BearerAuth: &controller.NewClientBearerAuthOpts{
				Token: sessionToken,
			},
			Id: methodId,
		})

		inputTemplateName := viper.GetString("template-id")
		templateInstance, err := cli.HandleTemplateSelection(cli.HandleTemplateSelectionOpts{
			Client:    client,
			UserInput: inputTemplateName,
			// ServiceLog: servicesLogs,
		})
		if err != nil {
			if errors.Is(err, cli.ErrorUserCancelled) {
				cli.PrintBoxedErrorMessage("We failed to get an input from you :/")
			}
			return fmt.Errorf("failed to select a template: %w", err)
		}

		o, _ := json.MarshalIndent(templateInstance, "", "  ")
		fmt.Println(string(o))

		// 1. trigger the POST /api/v1/automation endpoint

		// 2. a) endpoint returns a ${PENDING_AUTOMATION_ID} + list of variables and types

		// 2. a) 1) for each variable, ask user for it

		// 2. a) 2) run validations based on the type

		// 2. a) 3) trigger POST /api/v1/automation/${PENDING_AUTOMATION_ID} with variables

		// 2. b) 1) endpoint returns a ${PENDING_AUTOMATION_ID} and no variables

		// 2. b) 2) trigger POST /api/v1/automation/${PENDING_AUTOMATION_ID}

		// 3. display "successfully triggered message" to user along with instructions
		//    on how to view the status of the automation

		return nil
	},
}

func handleLocalExecution(cmd *cobra.Command, resourcePath string) error {
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
		return fmt.Errorf("automation execution failed with message: %w", err)
	}
	logsWaiter.Done()
	logsWaiter.Wait()
	return nil
}
