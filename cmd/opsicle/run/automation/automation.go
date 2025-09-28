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
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
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
		methodId := "opsicle/run/automation"
	enforceAuth:
		sessionToken, err := cli.RequireAuth(controllerUrl, methodId)
		if err != nil {
			rootCmd := cmd.Root()
			rootCmd.SetArgs([]string{"login"})
			_, err := rootCmd.ExecuteC()
			if err != nil {
				return err
			}
			goto enforceAuth
		}

		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			BearerAuth: &controller.NewClientBearerAuthOpts{
				Token: sessionToken,
			},
			Id: methodId,
		})
		if err != nil {
			return fmt.Errorf("failed to create controller client: %w", err)
		}

		inputTemplateReference := viper.GetString("template-id")
		templateInstance, err := cli.HandleTemplateSelection(cli.HandleTemplateSelectionOpts{
			Client:    client,
			UserInput: inputTemplateReference,
			// ServiceLog: servicesLogs,
		})
		if err != nil {
			if errors.Is(err, cli.ErrorUserCancelled) {
				cli.PrintBoxedErrorMessage("We failed to get an input from you :/")
			}
			return fmt.Errorf("failed to select a template: %w", err)
		}

		// 1. trigger the POST /api/v1/automation endpoint

		automationOutput, err := client.CreateAutomationV1(controller.CreateAutomationV1Input{
			TemplateId: templateInstance.Id,
		})
		if err != nil {
			return fmt.Errorf("failed to trigger automation: %w", err)
		}

		pendingAutomation := automationOutput.Data
		pendingAutomationId := pendingAutomation.AutomationId
		pendingAutomationInput := automationOutput.Data.VariableMap
		logrus.Debugf("received pending automation with id '%s'", pendingAutomationId)
		o, _ := json.MarshalIndent(automationOutput.Data, "", "  ")
		logrus.Debugf("received pending automation data:\n%s\n", string(o))
		var inputVariableMap map[string]any = nil

		if pendingAutomationInput != nil {
			logrus.Debugf("pending automation has variables to be filled up, creating form...")

			formFields := cli.FormFields{}
			variableMap := automationOutput.Data.VariableMap
			for id, variable := range variableMap {
				var defaultValue *string
				if variable.Default != nil {
					val := fmt.Sprintf("%v", variable.Default)
					defaultValue = &val
				}
				fieldType := cli.FormFieldString
				switch variable.Type {
				case "bool":
					fieldType = cli.FormFieldBoolean
				case "float":
					fieldType = cli.FormFieldFloat
				case "string":
					fieldType = cli.FormFieldString
				case "number":
					fieldType = cli.FormFieldInteger
				}
				input := cli.FormField{
					Id:          id,
					Label:       variable.Label,
					Description: variable.Description,
					Type:        fieldType,
				}
				if defaultValue != nil {
					input.DefaultValue = *defaultValue
				}
				formFields = append(formFields, input)
			}
			sort.Slice(formFields, func(i, j int) bool {
				return strings.Compare(formFields[i].Label, formFields[j].Label) < 0
			})
			variableInputForm := cli.CreateForm(cli.FormOpts{
				Title:       fmt.Sprintf("Executing template[%s]", automationOutput.Data.TemplateName),
				Description: "Please enter/confirm values for the following variables",
				Fields:      formFields,
			})
			if err := variableInputForm.GetInitWarnings(); err != nil {
				return fmt.Errorf("failed to create form as expected: %w", err)
			}

			logrus.Debugf("executing form to get input from user")

			formProgram := tea.NewProgram(variableInputForm)
			formOutput, err := formProgram.Run()
			if err != nil {
				return fmt.Errorf("failed to get user input: %w", err)
			}
			var ok bool
			variableInputForm, ok = formOutput.(*cli.FormModel)
			if !ok {
				return fmt.Errorf("failed to receive a cli.FormModel")
			}
			if errors.Is(variableInputForm.GetExitCode(), cli.ErrorUserCancelled) {
				fmt.Println("💬 Alrights, tell me again if you want to create an automation from this template")
				return errors.New("user cancelled action")
			}
			inputVariableMap = variableInputForm.GetValueMap()
			o, _ := json.MarshalIndent(inputVariableMap, "", "  ")
			logrus.Debugf("submitting variable map as follows:\n%s", string(o))
		}

		automationRunOutput, err := client.RunAutomationV1(controller.RunAutomationV1Input{
			AutomationId: automationOutput.Data.AutomationId,
			VariableMap:  inputVariableMap,
		})
		if err != nil {
			cli.PrintBoxedErrorMessage(
				fmt.Sprintf(
					"failed to run automation[%s] based on template[%s]: %s",
					automationOutput.Data.AutomationId,
					automationOutput.Data.TemplateId,
					err,
				),
			)
			return fmt.Errorf("failed to run automation")
		}

		// 4. display "successfully triggered message" to user along with instructions
		//    on how to view the status of the automation

		cli.PrintBoxedSuccessMessage(
			fmt.Sprintf(
				"Successfully triggered automation[%s] as automationRun[%s]",
				automationOutput.Data.AutomationId,
				automationRunOutput.Data.AutomationRunId,
			),
		)

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
