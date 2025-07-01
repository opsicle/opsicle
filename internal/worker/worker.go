package worker

import (
	"fmt"
	"net/url"
	"opsicle/internal/automations"
	"opsicle/internal/common"
	"opsicle/internal/config"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	ModeController string = "controller"
	ModeFilesystem string = "filesystem"
)

type Worker struct {
	// ControllerUrl defines the URL of the controller which
	// it should connect to for retrieving jobs, this is used
	// when the Mode is set to `ModeController`
	ControllerUrl string

	// DoneChannel will tell the worker to gracefully exit
	// when it's possible to do so
	DoneChannel chan common.Done

	// FilesystemPath defines the directory path where automation
	// manifests will be placed for execution, this is used when
	// the Mode is set to `ModeFilesystem`
	FilesystemPath string

	// ServiceLogs is a channel where service-level logs are emitted to
	ServiceLogs *chan LogEntry

	// AutomationLogs is a channel where logs from the executed automation are
	// emitted to
	AutomationLogs *chan string

	// PollInterval is the duration between polls of the queue
	PollInterval time.Duration

	// Runtime defines the runtime of the worker
	Runtime string

	// Mode which the worker should run in
	Mode string
}

func (w *Worker) Start() error {
	var serviceLogs chan LogEntry
	if w.ServiceLogs == nil {
		serviceLogs <- LogEntry{config.LogLevelWarn, "worker is starting with a noop service log, messages may be missed"}
		serviceLogs = make(chan LogEntry, 128)
		defer close(serviceLogs)
		go func() { // noop loop if log channel isn't specified
			for {
				_, ok := <-serviceLogs
				if !ok {
					return
				}
			}
		}()
	} else {
		serviceLogs = *w.ServiceLogs
	}
	var automationLogs chan string
	if w.AutomationLogs == nil {
		serviceLogs <- LogEntry{config.LogLevelWarn, "worker is starting with a noop automation log, messages may be missed"}
		automationLogs = make(chan string, 128)
		defer close(automationLogs)
		go func() { // noop loop if log channel isn't specified
			for {
				_, ok := <-automationLogs
				if !ok {
					return
				}
			}
		}()
	} else {
		automationLogs = *w.AutomationLogs
	}

	var lifecycleWaiter sync.WaitGroup
	lifecycleWaiter.Add(1)
	go func() {
		defer lifecycleWaiter.Done()
		serviceLogs <- LogEntry{config.LogLevelInfo, fmt.Sprintf("worker is starting in mode[%s] using runtime[%s]", w.Mode, w.Runtime)}
		switch w.Mode {
		case ModeController:
			if strings.Index(w.ControllerUrl, "://") < 0 {
				w.ControllerUrl = "http://" + w.ControllerUrl
			} else if strings.Index(w.ControllerUrl, "://") == 0 {
				w.ControllerUrl = "http" + w.ControllerUrl
			}
			controllerUrl, err := url.Parse(w.ControllerUrl)
			if err != nil {
				return
			}
			serviceLogs <- LogEntry{config.LogLevelInfo, fmt.Sprintf("starting polling from url[%s]", controllerUrl.String())}
			for {
				serviceLogs <- LogEntry{config.LogLevelDebug, fmt.Sprintf("querying url[%s] for new automations...", controllerUrl.String())}

				<-time.After(w.PollInterval)
			}
		case ModeFilesystem:
			directoryToWatch := w.FilesystemPath
			if !path.IsAbs(directoryToWatch) {
				baseDir := "/"
				if strings.Index(directoryToWatch, "~") == 0 {
					userHomeDir, err := os.UserHomeDir()
					if err != nil {
						return
					}
					baseDir = userHomeDir
				} else {
					workingDirectory, err := os.Getwd()
					if err != nil {
						return
					}
					baseDir = workingDirectory
				}
				directoryToWatch = filepath.Join(baseDir, directoryToWatch)
			}
			serviceLogs <- LogEntry{config.LogLevelInfo, fmt.Sprintf("using path[%s] as the queue", directoryToWatch)}
			directoryOfProcessedFiles := filepath.Join(directoryToWatch, "/.opsicle.done")
			if err := os.MkdirAll(directoryOfProcessedFiles, os.ModePerm); err != nil {
				serviceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("failed to ensure directory at path[%s]: %s", directoryOfProcessedFiles, err)}
				break
			}
			directoryOfProcessingFiles := filepath.Join(directoryToWatch, "/.opsicle.doing")
			if err := os.MkdirAll(directoryOfProcessingFiles, os.ModePerm); err != nil {
				serviceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("failed to ensure directory at path[%s]: %s", directoryOfProcessingFiles, err)}
				break
			}
			directoryOfLogs := filepath.Join(directoryToWatch, "/.opsicle.logs")
			if err := os.MkdirAll(directoryOfLogs, os.ModePerm); err != nil {
				serviceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("failed to ensure directory at path[%s]: %s", directoryOfProcessingFiles, err)}
				break
			}

			var ongoingAutomations sync.WaitGroup

			if err := startFilesystemQueueLoop(startFilesystemQueueLoopOpts{
				AutomationsCounter: &ongoingAutomations,
				Done:               w.DoneChannel,
				Handler: func(nextAutomation string, logsPath string) error {
					serviceLogs <- LogEntry{config.LogLevelDebug, fmt.Sprintf("loading automation from path[%s]...", nextAutomation)}
					automationInstance, err := automations.LoadAutomationFromFile(nextAutomation)
					if err != nil {
						serviceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("failed to load automation from path[%s]: %s", nextAutomation, err)}
						return fmt.Errorf("failed to load automation from path[%s]: %s", nextAutomation, err)
					}
					serviceLogs <- LogEntry{config.LogLevelDebug, fmt.Sprintf("running automation from path[%s]...", nextAutomation)}
					err = RunAutomation(RunAutomationOpts{
						Spec:           automationInstance,
						AutomationLogs: automationLogs,
						ServiceLogs:    serviceLogs,
					})
					if err != nil {
						serviceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("failed to run automation from path[%s]: %s", nextAutomation, err)}
						return fmt.Errorf("failed to run automation from path[%s]: %s", nextAutomation, err)
					}
					// for _, phase := range automationInstance.Spec.Phases {

					// }
					serviceLogs <- LogEntry{config.LogLevelDebug, fmt.Sprintf("successfully processed automation from path[%s]...", nextAutomation)}
					return nil
				},
				LogsPath:       directoryOfLogs,
				Path:           directoryToWatch,
				ProcessedPath:  directoryOfProcessedFiles,
				ProcessingPath: directoryOfProcessingFiles,
				PollInterval:   w.PollInterval,
				ServiceLogs:    serviceLogs,
			}); err != nil {
				serviceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("failure in execution loop: %s", err)}
			}

			ongoingAutomations.Wait()
		}
	}()

	lifecycleWaiter.Wait()
	return nil
}

type NewWorkerOpts struct {
	// AutomationLogs when defined will be the channel to which
	// **automation runtime** logs are emitted to
	AutomationLogs *chan string

	// DoneChannel will tell the worker to gracefully exit
	// when it's possible to do so
	DoneChannel chan common.Done

	// Mode defines the mode which the worker should run in
	Mode string

	// PollInterval is the duration between polls of the queue
	PollInterval time.Duration

	// Runtime defines the runtime of the worker
	Runtime string

	// ServiceLogs when defined will be the channel to which
	// **function** logs are emitted to
	ServiceLogs *chan LogEntry

	// Source defines the source path/url depending on the
	// mode the worker is running in
	Source string
}

func NewWorker(opts NewWorkerOpts) *Worker {
	worker := Worker{
		AutomationLogs: opts.AutomationLogs,
		DoneChannel:    opts.DoneChannel,
		ServiceLogs:    opts.ServiceLogs,
		Mode:           opts.Mode,
		PollInterval:   opts.PollInterval,
	}
	switch opts.Mode {
	case ModeController:
		worker.ControllerUrl = opts.Source
	case ModeFilesystem:
		worker.FilesystemPath = opts.Source
	}
	return &worker
}
