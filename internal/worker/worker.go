package worker

import (
	"fmt"
	"io/fs"
	"net/url"
	"opsicle/internal/automations"
	"opsicle/internal/config"
	"os"
	"path"
	"path/filepath"
	"sort"
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

	// Logs is a channel where logs are emitted to
	Logs *chan LogEntry

	// PollInterval is the duration between polls of the queue
	PollInterval time.Duration

	// Runtime defines the runtime of the worker
	Runtime string

	// Mode which the worker should run in
	Mode string
}

func (w *Worker) Start() error {
	var logChannel chan LogEntry
	if w.Logs == nil {
		logChannel = make(chan LogEntry, 128)
		defer close(logChannel)
		go func() { // noop loop if log channel isn't specified
			for {
				_, ok := <-logChannel
				if !ok {
					return
				}
			}
		}()
	} else {
		logChannel = *w.Logs
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
			filesystemPath := w.FilesystemPath
			if !path.IsAbs(filesystemPath) {
				cwd, err := os.Getwd()
				if err != nil {
					return
				}
				filesystemPath = path.Join(cwd, filesystemPath)
			}
			logrus.Infof("starting polling from path[%s]", filesystemPath)
			for {
				logrus.Tracef("checking path[%s] for new automations...", filesystemPath)
				latestFile, err := getLatestFile(filesystemPath)
				if err != nil {
					logChannel <- LogEntry{
						config.LogLevelError,
						fmt.Sprintf("failed to get the latest file: %s", err),
					}
				} else {
					logChannel <- LogEntry{
						config.LogLevelInfo,
						fmt.Sprintf("found file[%s], processing...", latestFile),
					}
					automationInstance, err := automations.LoadFromFile(latestFile)
					if err != nil {
						logChannel <- LogEntry{
							config.LogLevelError,
							fmt.Sprintf("failed to load automation from path[%s]: %s", latestFile, err),
						}
					} else {
						if err := RunAutomation(RunAutomationOpts{
							Spec: automationInstance,
							Logs: logChannel,
						}); err != nil {
							logChannel <- LogEntry{
								config.LogLevelError,
								fmt.Sprintf("failed to run automation from path[%s]: %s", latestFile, err),
							}
						} else {
							os.Remove(latestFile)
						}
					}
				}
				<-time.After(w.PollInterval)
			}
		}
	}()

	lifecycleWaiter.Wait()
	return nil
}

func getLatestFile(directoryPath string) (string, error) {
	var files []fs.FileInfo
	err := filepath.Walk(directoryPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		files = append(files, info)
		return nil
	})
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().After(files[j].ModTime())
	})

	latest := files[0]
	return filepath.Join(directoryPath, latest.Name()), nil
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
		Logs:         opts.Logs,
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
