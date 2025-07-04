package worker

import (
	"fmt"
	"opsicle/internal/common"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type startFilesystemQueueLoopOpts struct {
	AutomationsCounter *sync.WaitGroup
	Done               chan common.Done
	Handler            func(filePath string, logsPath string) error
	LogsPath           string
	Path               string
	ProcessingPath     string
	ProcessedPath      string
	PollInterval       time.Duration
	AutomationLogs     chan string
	ServiceLogs        chan common.ServiceLog
}

func startFilesystemQueueLoop(opts startFilesystemQueueLoopOpts) error {
	if opts.Path == "" {
		return fmt.Errorf("failed to get a path to use as a queue")
	}
	if opts.ProcessingPath == "" {
		return fmt.Errorf("failed to get a path to place files being processed in")
	}
	if opts.ProcessedPath == "" {
		return fmt.Errorf("failed to get a path to place processed files in")
	}
	isDone := false
	go func() {
		<-opts.Done
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "triggering exit sequence...")
		isDone = true
	}()
	for !isDone {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "checking path[%s] for new automations...", opts.Path)
		latestFilename, err := getLatestFilename(opts.Path)
		if err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to get the latest file: %s", err)
		} else {
			if latestFilename == "" {
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "no pending automations found at path[%s]", opts.Path)
			} else {
				currentFile := filepath.Join(opts.Path, latestFilename)
				fileToProcess := filepath.Join(opts.ProcessingPath, latestFilename)
				fileAfterProcessing := filepath.Join(opts.ProcessedPath, latestFilename)
				opts.AutomationsCounter.Add(1)
				if err := os.Rename(currentFile, fileToProcess); err != nil {
					opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to move file from path[%s] to path[%s]: %s", currentFile, fileToProcess, err)
					opts.AutomationsCounter.Done()
				} else {
					go func() {
						defer opts.AutomationsCounter.Done()
						if err := opts.Handler(fileToProcess, opts.LogsPath); err == nil {
							fmt.Println("no error")
							if err := os.Rename(fileToProcess, fileAfterProcessing); err != nil {
								opts.ServiceLogs <- common.ServiceLogf(common.LogLevelWarn, "failed to move file at path[%s] to path[%s]", fileToProcess, fileAfterProcessing)
							} else {
								opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "moved file at path[%s] to path[%s]", fileToProcess, fileAfterProcessing)
							}
						} else {
							fmt.Println("errored out")
							if err := os.Rename(fileToProcess, currentFile); err != nil {
								opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to move file from path[%s] to path[%s]: %s", fileToProcess, currentFile, err)
							}
						}
					}()
				}
			}
		}
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "waiting %v before next poll...", opts.PollInterval)
		<-time.After(opts.PollInterval)
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "exitted execution loop gracefully")
	return nil
}

func getLatestFilename(directoryPath string) (string, error) {
	entries, err := os.ReadDir(directoryPath)
	if err != nil {
		return "", err
	}

	var latestFile string
	var latestModTime time.Time

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().After(latestModTime) && strings.Index(entry.Name(), ".yaml") > 0 {
			latestModTime = info.ModTime()
			latestFile = entry.Name()
		}
	}

	if latestFile == "" {
		return "", nil
	}

	return latestFile, nil
}
