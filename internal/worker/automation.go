package worker

import (
	"context"
	"fmt"
	"io"
	"opsicle/internal/automations"
	"opsicle/internal/config"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

type RunAutomationOpts struct {
	Spec *automations.Automation
	Logs chan LogEntry
	Done *chan struct{}
}

func RunAutomation(opts RunAutomationOpts) error {
	var doneChannel chan struct{}
	if opts.Done != nil {
		doneChannel = *opts.Done
	} else {
		doneChannel = make(chan struct{})
		go func() { // noop doneChannel
			defer close(doneChannel)
			_, ok := <-doneChannel
			if !ok {
				return
			}
		}()
	}
	defer func() {
		doneChannel <- struct{}{}
	}()
	phases := serializePhases(opts.Spec.Spec.Phases)
	volumeMounts, err := serializeVolumeMounts(opts.Spec.Spec.VolumeMounts)
	if err != nil {
		return fmt.Errorf("failed to process volume mounts: %s", err)
	}

	if err := runAutomation(automationSpec{
		Phases:       phases,
		VolumeMounts: volumeMounts,
		Logs:         opts.Logs,
	}); err != nil {
		return fmt.Errorf("failed to run automation: %s", err)
	}

	return nil
}

func runAutomation(spec automationSpec) error {
	baseCtx := context.Background()
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithVersion("1.49"),
	)
	if err != nil {
		return err
	}

	var mounts []mount.Mount
	for _, vm := range spec.VolumeMounts {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: vm.HostPath,
			Target: vm.ContainerPath,
		})
	}

	for _, phase := range spec.Phases {
		spec.Logs <- LogEntry{config.LogLevelInfo, fmt.Sprintf("==> Starting phase: %s\n", phase.Name)}

		timeout := time.Duration(phase.Timeout) * time.Second
		ctx, cancel := context.WithTimeout(baseCtx, timeout)
		defer cancel()

		_, err := cli.ImagePull(ctx, phase.Image, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("failed to pull image %s: %w", phase.Image, err)
		}

		resp, err := cli.ContainerCreate(ctx, &container.Config{
			Image: phase.Image,
			Cmd:   []string{"sh", "-c", strings.Join(phase.Commands, " && ")},
			Tty:   false,
		}, &container.HostConfig{
			Mounts: mounts,
		}, nil, nil, "")
		if err != nil {
			return err
		}

		if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			return err
		}

		logChan := make(chan LogEntry)
		done := make(chan struct{})
		go streamDockerLogs(ctx, cli, resp.ID, logChan, done)

		waitCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

		var waiter sync.WaitGroup
		waiter.Add(1)
		go func() {
			defer waiter.Done()
			spec.Logs <- LogEntry{config.LogLevelInfo, fmt.Sprintf("started streaming logs for phase[%s]", phase.Name)}
			for {
				logLine, ok := <-logChan
				if !ok {
					break
				}
				fmt.Printf(logLine.Message)
			}
		}()
		waiter.Add(1)
		go func() {
			defer waiter.Done()
			isDone := false
			for {
				if isDone {
					break
				}
				select {
				case <-ctx.Done():
					cli.ContainerKill(context.Background(), resp.ID, "SIGKILL")
					logChan <- LogEntry{config.LogLevelError, fmt.Sprintf("phase[%s] timed out", phase.Name)}
				case err := <-errCh:
					if err != nil {
						logChan <- LogEntry{config.LogLevelError, fmt.Sprintf("error: %s", err)}
					}
				case <-done:
					isDone = true
				case status := <-waitCh:
					if status.StatusCode != 0 {
						logChan <- LogEntry{config.LogLevelError, fmt.Sprintf("container exited with status %d", status.StatusCode)}
						isDone = true
					}
				}
			}
			spec.Logs <- LogEntry{config.LogLevelInfo, fmt.Sprintf("closing logs channel for phase[%s]", phase.Name)}
			close(logChan)
		}()
		waiter.Wait()
	}

	return nil
}

func streamDockerLogs(ctx context.Context, cli *client.Client, containerId string, logChan chan<- LogEntry, done chan<- struct{}) {
	outputStream, err := cli.ContainerLogs(
		ctx,
		containerId,
		container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		},
	)
	if err != nil {
		logChan <- LogEntry{config.LogLevelError, fmt.Sprintf("failed to stream logs: %v", err)}
		done <- struct{}{}
		return
	}
	defer outputStream.Close()

	buf := make([]byte, 1024)
	for {
		n, err := outputStream.Read(buf)
		if n > 0 {
			logChan <- LogEntry{config.LogLevelInfo, string(buf[:n])}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			logChan <- LogEntry{config.LogLevelError, fmt.Sprintf("log read error: %v", err)}
		}
	}
	done <- struct{}{}
}
