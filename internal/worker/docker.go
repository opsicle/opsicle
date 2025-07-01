package worker

import (
	"context"
	"fmt"
	"io"
	"opsicle/internal/common"
	"opsicle/internal/config"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// streamDockerLogsOpts provides options for the streamDockerLogs
// static method in this package
type streamDockerLogsOpts struct {
	// BufferSize determines the size of the logs buffer for reading
	// output from the container identified by `.ContainerId`. Increase
	// this if the container outputs logs greater than `DefaultBufferSize`
	BufferSize *int

	// ContainerId is the ID of the Docker container to stream logs from
	ContainerId string

	// AutomationLogs is for the caller to receive logs from the
	// **container**, NOT the function
	AutomationLogs chan string

	// Context is for inheritance of the caller's context
	Context *context.Context

	// ServiceLogs is for the caller to receive logs from the
	// **function**, NOT the container
	ServiceLogs chan LogEntry

	// DockerClient is the client which is able to perform Docker operations
	// on the Docker daemon running locally
	DockerClient *client.Client

	// DoneChannel is used by the function to indicate that it is done with
	// streaming logs
	DoneChannel chan<- common.Done

	// IsStderrEnabled defines whether stderr should be captured, defaults to
	// `DefaultIsStderrEnabled` when not defined
	IsStderrEnabled *bool

	// IsStdoutEnabled defines whether stdout should be captured, defaults to
	// `DefaultIsStdoutEnabled` when not defined
	IsStdoutEnabled *bool
}

func streamDockerLogs(opts streamDockerLogsOpts) error {
	var ctx context.Context
	if opts.Context == nil {
		ctx = context.Background()
	} else {
		ctx = *opts.Context
	}

	bufferSize := DefaultBufferSize
	if opts.BufferSize != nil {
		bufferSize = *opts.BufferSize
	}

	isStderrEnabled := DefaultIsStderrEnabled
	if opts.IsStderrEnabled != nil {
		isStderrEnabled = *opts.IsStderrEnabled
	}
	isStdoutEnabled := DefaultIsStdoutEnabled
	if opts.IsStdoutEnabled != nil {
		isStdoutEnabled = *opts.IsStdoutEnabled
	}
	displayContainerId := opts.ContainerId[:11]

	opts.ServiceLogs <- LogEntry{config.LogLevelDebug, fmt.Sprintf("container[%s]: creating container logs stream...", displayContainerId)}
	out, err := opts.DockerClient.ContainerLogs(
		ctx, opts.ContainerId, container.LogsOptions{
			ShowStdout: isStdoutEnabled,
			ShowStderr: isStderrEnabled,
			Follow:     true,
		})
	if err != nil {
		opts.ServiceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("container[%s]: failed to stream logs: %v", displayContainerId, err)}
		opts.DoneChannel <- common.Done{}
		return nil
	}
	defer out.Close()

	outReader, outWriter := io.Pipe()
	errReader, errWriter := io.Pipe()

	go func() {
		if _, err := stdcopy.StdCopy(outWriter, errWriter, out); err != nil {
			opts.ServiceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("container[%s]: failed to demux stdout/sterr stream: %v", displayContainerId, err)}
		}
		outWriter.Close()
		errWriter.Close()
	}()

	var logStreamWaiter sync.WaitGroup
	logStreamWaiter.Add(1)
	go func() {
		defer logStreamWaiter.Done()
		buffer := make([]byte, bufferSize)
		opts.ServiceLogs <- LogEntry{config.LogLevelDebug, fmt.Sprintf("container[%s]: created stdout stream with bufferSize[%v]", displayContainerId, bufferSize)}
		for {
			n, err := outReader.Read(buffer)
			if n > 0 {
				opts.ServiceLogs <- LogEntry{config.LogLevelTrace, fmt.Sprintf("container[%s]: streamed %v bytes from stdout", displayContainerId, n)}
				opts.AutomationLogs <- string(buffer[:n])
			}
			if err != nil {
				if err != io.EOF {
					opts.ServiceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("container[%s]: received error while streaming stdout: %s", displayContainerId, err)}
					break
				}
				opts.ServiceLogs <- LogEntry{config.LogLevelTrace, fmt.Sprintf("container[%s]: eof received on stdout", displayContainerId)}
				break
			}
		}
		opts.ServiceLogs <- LogEntry{config.LogLevelTrace, fmt.Sprintf("container[%s]: stdout stream closed", displayContainerId)}
	}()

	logStreamWaiter.Add(1)
	go func() {
		defer logStreamWaiter.Done()
		buffer := make([]byte, bufferSize)
		opts.ServiceLogs <- LogEntry{config.LogLevelDebug, fmt.Sprintf("container[%s]: created stderr stream with bufferSize[%v]", displayContainerId, bufferSize)}
		for {
			n, err := errReader.Read(buffer)
			if n > 0 {
				opts.ServiceLogs <- LogEntry{config.LogLevelTrace, fmt.Sprintf("container[%s]: streamed %v bytes from stderr", displayContainerId, n)}
				opts.AutomationLogs <- prefixWithStderr(string(buffer[:n]))
			}
			if err != nil {
				if err != io.EOF {
					opts.ServiceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("container[%s]: received error while streaming stderr: %s", displayContainerId, err)}
					break
				}
				opts.ServiceLogs <- LogEntry{config.LogLevelTrace, fmt.Sprintf("container[%s]: eof received on stderr", displayContainerId)}
				break
			}
		}
		opts.ServiceLogs <- LogEntry{config.LogLevelTrace, fmt.Sprintf("container[%s]: stderr stream closed", displayContainerId)}
	}()

	logStreamWaiter.Wait()
	opts.ServiceLogs <- LogEntry{config.LogLevelTrace, fmt.Sprintf("container[%s]: logs streaming is complete", displayContainerId)}
	opts.DoneChannel <- common.Done{}
	return nil
}

func prefixWithStderr(text string) string {
	return "[stderr] " + text
}

// // func streamDockerLogs(ctx context.Context, cli *client.Client, containerId string, logChan chan<- LogEntry, done chan<- struct{}) {
// func streamDockerLogs(opts streamDockerLogsOpts) error {
// 	var ctx context.Context
// 	if opts.Context == nil {
// 		ctx = context.Background()
// 	} else {
// 		ctx = *opts.Context
// 	}

// 	bufferSize := DefaultBufferSize
// 	if opts.BufferSize != nil {
// 		bufferSize = *opts.BufferSize
// 	}

// 	// isStderrEnabled := DefaultIsStderrEnabled
// 	// if opts.IsStderrEnabled != nil {
// 	// 	isStderrEnabled = *opts.IsStderrEnabled
// 	// }
// 	isStdoutEnabled := DefaultIsStdoutEnabled
// 	if opts.IsStdoutEnabled != nil {
// 		isStdoutEnabled = *opts.IsStdoutEnabled
// 	}

// 	displayContainerId := opts.ContainerId[:11]

// 	defer func() {
// 		<-time.After(100 * time.Millisecond)
// 		opts.DoneChannel <- common.Done{}
// 	}()
// 	opts.ServiceLogs <- LogEntry{config.LogLevelDebug, fmt.Sprintf("container[%s]: establishing streaming with container logs...", displayContainerId)}
// 	outputStream, err := opts.DockerClient.ContainerLogs(
// 		ctx,
// 		opts.ContainerId,
// 		container.LogsOptions{
// 			ShowStdout: isStdoutEnabled,
// 			// ShowStderr: isStderrEnabled,
// 			Follow: true,
// 		},
// 	)
// 	if err != nil {
// 		return fmt.Errorf("failed to stream logs for container[%s]: %v", displayContainerId, err)
// 	}
// 	opts.ServiceLogs <- LogEntry{config.LogLevelInfo, fmt.Sprintf("container[%s]: successfully established streaming with container logs", displayContainerId)}
// 	defer outputStream.Close()

// 	for {
// 		outputBuffer := make([]byte, bufferSize)
// 		n, err := outputStream.Read(outputBuffer)
// 		if n > 0 {
// 			opts.ServiceLogs <- LogEntry{config.LogLevelTrace, fmt.Sprintf("container[%s]: received %v bytes of logs", displayContainerId, n)}
// 			fmt.Printf("sending %v bytes:\n------START------\n%s\n------END------\n", n, string(outputBuffer))
// 			opts.AutomationLogs <- string(outputBuffer[:n])
// 			opts.ServiceLogs <- LogEntry{config.LogLevelTrace, fmt.Sprintf("container[%s]: emitted %v bytes of logs", displayContainerId, n)}
// 		}
// 		if err != nil {
// 			if err == io.EOF {
// 				opts.ServiceLogs <- LogEntry{config.LogLevelTrace, fmt.Sprintf("container[%s]: received eof from docker daemon, exiting gracefully...", displayContainerId)}
// 				break
// 			}
// 			return fmt.Errorf("log read error in container[%s]: %v", displayContainerId, err)
// 		}
// 		<-time.After(10 * time.Millisecond)
// 	}

// 	return nil
// }
