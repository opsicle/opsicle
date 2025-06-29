package worker

import (
	"fmt"
	"opsicle/internal/automationtemplates"
	"os"
	"path"
)

type LogEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

type volumeMount struct {
	HostPath      string `json:"hostPath" yaml:"hostPath"`
	ContainerPath string `json:"containerPath" yaml:"containerPath"`
}

type automationPhase struct {
	Name     string   `json:"name" yaml:"name"`
	Image    string   `json:"image" yaml:"image"`
	Commands []string `json:"commands" yaml:"commands"`
	Timeout  int      `json:"timeout" yaml:"timeout"`
}

func serializePhases(phases []automationtemplates.Phase) []automationPhase {
	output := []automationPhase{}
	for _, phase := range phases {
		timeout := phase.Timeout
		if timeout < 1 {
			timeout = 60
		}
		output = append(output, automationPhase{
			Name:     phase.Name,
			Image:    phase.Image,
			Commands: phase.Commands,
			Timeout:  timeout,
		})
	}
	return output
}

type automationSpec struct {
	Phases       []automationPhase `json:"phases" yaml:"phases"`
	VolumeMounts []volumeMount     `json:"volumeMounts" yaml:"volumeMounts"`

	AutomationLogs chan string   `json:"-"`
	ServiceLogs    chan LogEntry `json:"-"`
}

func serializeVolumeMounts(volumeMounts []automationtemplates.VolumeMount) ([]volumeMount, error) {
	output := []volumeMount{}
	for _, volumeMountInstance := range volumeMounts {
		hostPath := volumeMountInstance.Host
		if !path.IsAbs(hostPath) {
			currentDirectory, err := os.Getwd()
			if err != nil {
				return nil, fmt.Errorf("failed to process volume mount using a relative path and couldn't retrieve current working directory: %s", err)
			}
			hostPath = path.Join(currentDirectory, hostPath)
		}
		output = append(output, volumeMount{
			HostPath:      hostPath,
			ContainerPath: volumeMountInstance.Container,
		})
	}
	return output, nil
}
