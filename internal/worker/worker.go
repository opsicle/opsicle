package worker

import (
	"fmt"
	"net/url"
	"opsicle/internal/automations"
	"opsicle/internal/config"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
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
		logrus.Infof("worker is starting in mode[%s] using runtime[%s]", w.Mode, w.Runtime)
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
			logrus.Infof("starting polling from url[%s]", controllerUrl.String())
			for {
				logrus.Tracef("querying url[%s] for new automations...", controllerUrl.String())

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
			logrus.Infof("using path[%s] as the queue", directoryToWatch)
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

			go func() {
				for {
					automationLog, ok := <-automationLogs
					if !ok {
						break
					}
					fmt.Print(automationLog)
				}
			}()

			startFilesystemQueueLoop(startFilesystemQueueLoopOpts{
				Handler: func(nextAutomation string) error {
					automationInstance, err := automations.LoadFromFile(nextAutomation)
					if err != nil {
						serviceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("failed to load automation from path[%s]: %s", nextAutomation, err)}
						return fmt.Errorf("failed to load automation from path[%s]: %s", nextAutomation, err)
					}
					err = RunAutomation(RunAutomationOpts{
						Spec:           automationInstance,
						AutomationLogs: automationLogs,
						ServiceLogs:    serviceLogs,
					})
					if err != nil {
						serviceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("failed to run automation from path[%s]: %s", nextAutomation, err)}
						return fmt.Errorf("failed to run automation from path[%s]: %s", nextAutomation, err)
					}
					return nil
				},
				Path:           directoryToWatch,
				ProcessedPath:  directoryOfProcessedFiles,
				ProcessingPath: directoryOfProcessingFiles,
				PollInterval:   w.PollInterval,
				ServiceLogs:    serviceLogs,
			})
		}
	}()

	lifecycleWaiter.Wait()
	return nil
}

type NewWorkerOpts struct {
	// Mode defines the mode which the worker should run in
	Mode string

	// PollInterval is the duration between polls of the queue
	PollInterval time.Duration

	// Logs when defined will be the channel to which logs are
	// emitted to
	Logs *chan LogEntry

	// Runtime defines the runtime of the worker
	Runtime string

	// Source defines the source path/url depending on the
	// mode the worker is running in
	Source string
}

func NewWorker(opts NewWorkerOpts) *Worker {
	worker := Worker{
		ServiceLogs:  opts.Logs,
		Mode:         opts.Mode,
		PollInterval: opts.PollInterval,
	}
	switch opts.Mode {
	case ModeController:
		worker.ControllerUrl = opts.Source
	case ModeFilesystem:
		worker.FilesystemPath = opts.Source
	}
	return &worker
}
