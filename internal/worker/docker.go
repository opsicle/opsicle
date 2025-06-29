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

	// AutomationLogsChannel is for the caller to receive logs from the
	// **container**, NOT the function. WARNING: this channel will be
	// closed by this function upon completion
	AutomationLogsChannel chan string

	// Context is for inheritance of the caller's context
	Context *context.Context

	// ControllerLogsChannel is for the caller to receive logs from the
	// **function**, NOT the container
	ControllerLogsChannel chan LogEntry

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

	out, err := opts.DockerClient.ContainerLogs(
		ctx, opts.ContainerId, container.LogsOptions{
			ShowStdout: true, ShowStderr: true, Follow: true,
		})
	if err != nil {
		opts.ControllerLogsChannel <- LogEntry{config.LogLevelError, fmt.Sprintf("failed to stream logs: %v", err)}
		opts.DoneChannel <- common.Done{}
		return nil
	}
	defer out.Close()

	rOut, wOut := io.Pipe()
	rErr, wErr := io.Pipe()

	go func() {
		stdcopy.StdCopy(wOut, wErr, out)
		wOut.Close()
		wErr.Close()
	}()

	var waiter sync.WaitGroup
	waiter.Add(1)
	go func() {
		defer waiter.Done()
		buf := make([]byte, 1024)
		for {
			n, err := rOut.Read(buf)
			if n > 0 {
				opts.AutomationLogsChannel <- string(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	waiter.Add(1)
	go func() {
		defer waiter.Done()
		buf := make([]byte, 1024)
		for {
			n, err := rErr.Read(buf)
			if n > 0 {
				opts.AutomationLogsChannel <- string(buf[:n]) // Optionally prefix with [stderr]
			}
			if err != nil {
				break
			}
		}
		opts.DoneChannel <- common.Done{}
	}()

	waiter.Wait()
	close(opts.AutomationLogsChannel)
	return nil
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
// 	opts.ControllerLogsChannel <- LogEntry{config.LogLevelDebug, fmt.Sprintf("container[%s]: establishing streaming with container logs...", displayContainerId)}
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
// 	opts.ControllerLogsChannel <- LogEntry{config.LogLevelInfo, fmt.Sprintf("container[%s]: successfully established streaming with container logs", displayContainerId)}
// 	defer outputStream.Close()

// 	for {
// 		outputBuffer := make([]byte, bufferSize)
// 		n, err := outputStream.Read(outputBuffer)
// 		if n > 0 {
// 			opts.ControllerLogsChannel <- LogEntry{config.LogLevelTrace, fmt.Sprintf("container[%s]: received %v bytes of logs", displayContainerId, n)}
// 			fmt.Printf("sending %v bytes:\n------START------\n%s\n------END------\n", n, string(outputBuffer))
// 			opts.AutomationLogsChannel <- string(outputBuffer[:n])
// 			opts.ControllerLogsChannel <- LogEntry{config.LogLevelTrace, fmt.Sprintf("container[%s]: emitted %v bytes of logs", displayContainerId, n)}
// 		}
// 		if err != nil {
// 			if err == io.EOF {
// 				opts.ControllerLogsChannel <- LogEntry{config.LogLevelTrace, fmt.Sprintf("container[%s]: received eof from docker daemon, exiting gracefully...", displayContainerId)}
// 				break
// 			}
// 			return fmt.Errorf("log read error in container[%s]: %v", displayContainerId, err)
// 		}
// 		<-time.After(10 * time.Millisecond)
// 	}

// 	return nil
// }
