package worker

import (
	"context"
	"fmt"
	"opsicle/internal/automations"
	"opsicle/internal/common"
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
	DockerApiVersion *string
	Spec             *automations.Automation
	AutomationLogs   chan string
	ServiceLogs      chan LogEntry
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
	phases := serializePhases(opts.Spec.Spec.Phases)
	if len(phases) == 0 {
		return fmt.Errorf("failed to receive a phase, at least one phase needs to be present")
	}
	volumeMounts, err := serializeVolumeMounts(opts.Spec.Spec.VolumeMounts)
	if err != nil {
		return fmt.Errorf("failed to process volume mounts: %s", err)
	}
	dockerApiVersion := DefaultDockerApiVersion
	if opts.DockerApiVersion != nil {
		dockerApiVersion = *opts.DockerApiVersion
	}

	if err := runAutomation(
		automationSpec{
			Phases:         phases,
			VolumeMounts:   volumeMounts,
			AutomationLogs: opts.AutomationLogs,
			ServiceLogs:    opts.ServiceLogs,
		},
		runAutomationOpts{
			DockerApiVersion: dockerApiVersion,
		},
	); err != nil {
		return fmt.Errorf("failed to run automation: %s", err)
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
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: vm.HostPath,
			Target: vm.ContainerPath,
		})
	}

	for _, phase := range spec.Phases {
		spec.ServiceLogs <- LogEntry{config.LogLevelInfo, fmt.Sprintf("phase[%s]: starting", phase.Name)}

		timeout := time.Duration(phase.Timeout) * time.Second
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
		go func() {
			if err := streamDockerLogs(streamDockerLogsOpts{
				ContainerId:           containerInfo.ID,
				ControllerLogsChannel: spec.ServiceLogs,
				AutomationLogsChannel: containerLogs,
				DockerClient:          dockerClient,
				DoneChannel:           done,
			}); err != nil {

			}
		}()

		containerResponses, containerErrors := dockerClient.ContainerWait(
			phaseCtx,
			containerInfo.ID,
			container.WaitConditionNotRunning,
		)

		var waiter sync.WaitGroup
		waiter.Add(1)
		go func() {
			defer waiter.Done()
			spec.ServiceLogs <- LogEntry{config.LogLevelInfo, fmt.Sprintf("phase[%s]: started streaming logs for container[%s]", phase.Name, displayContainerId)}
			for {
				containerLog, ok := <-containerLogs
				if !ok {
					break
				}
				spec.AutomationLogs <- string(containerLog)
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
				case <-phaseCtx.Done():
					spec.ServiceLogs <- LogEntry{config.LogLevelWarn, fmt.Sprintf("phase[%s]: timed out, killing container[%s]...", phase.Name, displayContainerId)}
					if err := dockerClient.ContainerKill(context.Background(), containerInfo.ID, "SIGKILL"); err != nil {
						spec.ServiceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("phase[%s]: timed out but container[%s] failed to be killed", phase.Name, displayContainerId)}
					} else {
						spec.ServiceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("phase[%s] timed out and container[%s] was killed", phase.Name, displayContainerId)}
					}
				case err := <-containerErrors:
					if err != nil {
						spec.ServiceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("container[%s]: encountered error: %s", displayContainerId, err)}
					}
				case <-done:
					isDone = true
				case status := <-containerResponses:
					if status.StatusCode != 0 {
						spec.ServiceLogs <- LogEntry{config.LogLevelError, fmt.Sprintf("container[%s]: exited with status %d", displayContainerId, status.StatusCode)}
						isDone = true
					}
				}
			}
			spec.ServiceLogs <- LogEntry{config.LogLevelInfo, fmt.Sprintf("phase[%s]: container[%s] is done", phase.Name, displayContainerId)}
			time.After(100 * time.Second)
		}()
		waiter.Wait()
	}

	return nil
}
