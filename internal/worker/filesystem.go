package worker

import (
	"fmt"
	"opsicle/internal/common"
	"opsicle/internal/config"
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
	ServiceLogs        chan LogEntry
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
		opts.ServiceLogs <- LogEntry{config.LogLevelInfo, "triggering exit sequence..."}
		isDone = true
	}()
	for !isDone {
		opts.ServiceLogs <- LogEntry{config.LogLevelTrace, fmt.Sprintf("checking path[%s] for new automations...", opts.Path)}
		latestFilename, err := getLatestFilename(opts.Path)
		if err != nil {
			opts.ServiceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("failed to get the latest file: %s", err)}
		} else {
			if latestFilename == "" {
				opts.ServiceLogs <- LogEntry{config.LogLevelInfo, fmt.Sprintf("no pending automations found at path[%s]", opts.Path)}
			} else {
				currentFile := filepath.Join(opts.Path, latestFilename)
				fileToProcess := filepath.Join(opts.ProcessingPath, latestFilename)
				fileAfterProcessing := filepath.Join(opts.ProcessedPath, latestFilename)
				opts.AutomationsCounter.Add(1)
				if err := os.Rename(currentFile, fileToProcess); err != nil {
					opts.ServiceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("failed to move file from path[%s] to path[%s]: %s", currentFile, fileToProcess, err)}
					opts.AutomationsCounter.Done()
				} else {
					go func() {
						defer opts.AutomationsCounter.Done()
						if err := opts.Handler(fileToProcess, opts.LogsPath); err == nil {
							fmt.Println("no error")
							if err := os.Rename(fileToProcess, fileAfterProcessing); err != nil {
								opts.ServiceLogs <- LogEntry{config.LogLevelWarn, fmt.Sprintf("failed to move file at path[%s] to path[%s]", fileToProcess, fileAfterProcessing)}
							} else {
								opts.ServiceLogs <- LogEntry{config.LogLevelInfo, fmt.Sprintf("moved file at path[%s] to path[%s]", fileToProcess, fileAfterProcessing)}
							}
						} else {
							fmt.Println("errored out")
							if err := os.Rename(fileToProcess, currentFile); err != nil {
								opts.ServiceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("failed to move file from path[%s] to path[%s]: %s", fileToProcess, currentFile, err)}
							}
						}
					}()
				}
			}
		}
		opts.ServiceLogs <- LogEntry{config.LogLevelDebug, fmt.Sprintf("waiting %v before next poll...", opts.PollInterval)}
		<-time.After(opts.PollInterval)
	}
	opts.ServiceLogs <- LogEntry{config.LogLevelInfo, "exitted execution loop gracefully"}
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
