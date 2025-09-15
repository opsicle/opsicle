package worker

import (
	"context"
	"fmt"
	"opsicle/internal/automations"
	"opsicle/internal/common"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

type RunAutomationOpts struct {
	DockerApiVersion *string
	Spec             *automations.Automation
	AutomationLogs   chan string
	ServiceLogs      chan common.ServiceLog
	Done             *chan common.Done
}

func RunAutomation(opts RunAutomationOpts) error {
	var doneChannel chan common.Done
	if opts.Done != nil {
		doneChannel = *opts.Done
	} else {
		doneChannel = make(chan common.Done)
		go func() { // noop doneChannel
			<-doneChannel
			close(doneChannel)
		}()
	}
	defer func() {
		doneChannel <- struct{}{}
	}()
	if opts.Spec.Metadata.Name == "" {
		return fmt.Errorf("failed to receive a name, the name needs to be defined")
	}
	dockerApiVersion := DefaultDockerApiVersion
	if opts.DockerApiVersion != nil {
		dockerApiVersion = *opts.DockerApiVersion
	}

	if err := runAutomation(
		automationSpec{
			Phases:         opts.Spec.Spec.Phases,
			VolumeMounts:   opts.Spec.Spec.VolumeMounts,
			AutomationLogs: opts.AutomationLogs,
			ServiceLogs:    opts.ServiceLogs,
		},
		runAutomationOpts{
			DockerApiVersion: dockerApiVersion,
		},
	); err != nil {
		return fmt.Errorf("failed to run automation: %w", err)
	}

	return nil
}

type runAutomationOpts struct {
	DockerApiVersion string
}

func runAutomation(spec automationSpec, opts runAutomationOpts) error {
	baseCtx := context.Background()
	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithVersion(opts.DockerApiVersion),
	)
	if err != nil {
		return err
	}

	var mounts []mount.Mount
	for _, vm := range spec.VolumeMounts {
		hostVolumePath := vm.Host
		if !path.IsAbs(hostVolumePath) {
			workingDirectory, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
			hostVolumePath = filepath.Join(workingDirectory, hostVolumePath)
		}
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: hostVolumePath,
			Target: vm.Container,
		})
	}

	for _, phase := range spec.Phases {
		spec.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "phase[%s]: starting", phase.Name)

		var timeout time.Duration
		if phase.Timeout == 0 {
			timeout = 60 * time.Second
		} else {
			timeout = time.Duration(phase.Timeout) * time.Second
		}
		phaseCtx, cancel := context.WithTimeout(baseCtx, timeout)
		defer cancel()

		_, err := dockerClient.ImagePull(phaseCtx, phase.Image, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("failed to pull image %s: %w", phase.Image, err)
		}

		containerInfo, err := dockerClient.ContainerCreate(phaseCtx, &container.Config{
			Image: phase.Image,
			Cmd:   []string{"sh", "-c", strings.Join(phase.Commands, " && ")},
			Tty:   false,
		}, &container.HostConfig{
			Mounts: mounts,
		}, nil, nil, "")
		if err != nil {
			return err
		}

		displayContainerId := containerInfo.ID[:11]

		if err := dockerClient.ContainerStart(phaseCtx, containerInfo.ID, container.StartOptions{}); err != nil {
			return err
		}

		containerLogs := make(chan string, 128)
		done := make(chan common.Done)
		dockerLogStreamingOpts := streamDockerLogsOpts{
			ContainerId:    containerInfo.ID,
			ServiceLogs:    spec.ServiceLogs,
			AutomationLogs: containerLogs,
			DockerClient:   dockerClient,
			DoneChannel:    done,
		}
		go func(dockerLogStreamingOpts streamDockerLogsOpts) {
			if err := streamDockerLogs(dockerLogStreamingOpts); err != nil {
				spec.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "phase[%s]: failed to stream logs for container[%s]: %s", phase.Name, displayContainerId, err)
			}
		}(dockerLogStreamingOpts)

		containerResponses, containerErrors := dockerClient.ContainerWait(
			phaseCtx,
			containerInfo.ID,
			container.WaitConditionNotRunning,
		)

		var waiter sync.WaitGroup
		waiter.Add(1)
		go func() {
			defer waiter.Done()
			spec.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "phase[%s]: started streaming logs for container[%s]", phase.Name, displayContainerId)
			var phaseLogsMutex sync.Mutex
			for {
				containerLog, ok := <-containerLogs
				if !ok {
					break
				}
				spec.AutomationLogs <- containerLog
				phaseLogsMutex.Lock()
				phase.Logs = append(phase.Logs, automations.PhaseLog{
					Timestamp: time.Now().Format("2006-01-02T15:04:05"),
					Message:   containerLog,
				})
				phaseLogsMutex.Unlock()
			}
		}()
		waiter.Add(1)
		go func() {
			defer waiter.Done()
			isDone := false
			for !isDone {
				select {
				case <-phaseCtx.Done():
					spec.ServiceLogs <- common.ServiceLogf(common.LogLevelWarn, "phase[%s]: timed out, killing container[%s]...", phase.Name, displayContainerId)
					if err := dockerClient.ContainerKill(context.Background(), containerInfo.ID, "SIGKILL"); err != nil {
						spec.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "phase[%s]: timed out but container[%s] failed to be killed", phase.Name, displayContainerId)
					} else {
						spec.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "phase[%s] timed out and container[%s] was killed", phase.Name, displayContainerId)
					}
				case err := <-containerErrors:
					if err != nil {
						spec.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "container[%s]: encountered error: %s", displayContainerId, err)
					}
				case <-done:
					isDone = true
				case status := <-containerResponses:
					if status.StatusCode != 0 {
						spec.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "container[%s]: exited with status %d", displayContainerId, status.StatusCode)
						isDone = true
					}
				}
			}
			spec.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "phase[%s]: container[%s] is done", phase.Name, displayContainerId)
			<-time.After(1 * time.Second)
			close(containerLogs)
		}()
		waiter.Wait()
	}

	return nil
}
