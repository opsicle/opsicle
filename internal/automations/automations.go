package automations

import (
	"opsicle/internal/common"
	"os"

	"gopkg.in/yaml.v3"
)

type Automation struct {
	common.Resource `json:"resource" yaml:",inline"`
	Spec            AutomationSpec `json:"spec" yaml:"spec"`
}

type AutomationSpec struct {
	// VolumeMounts defines any volume mounts in play when containers are
	// spun up
	VolumeMounts []VolumeMount `json:"volumeMounts" yaml:"volumeMounts"`

	// Phases defines the various steps of the automation
	Phases []Phase `json:"phases" yaml:"phases"`
}

type VolumeMount struct {
	Host      string `json:"host" yaml:"host"`
	Container string `json:"container" yaml:"container"`
}

type Phase struct {
	Name     string    `json:"name" yaml:"name"`
	Image    string    `json:"image" yaml:"image"`
	Commands []string  `json:"command" yaml:"commands"`
	Timeout  int       `json:"timeout" yaml:"timeout"`
	Logs     PhaseLogs `json:"logs,omitempty" yaml:"logs"`
}

type PhaseLogs []PhaseLog

type PhaseLog struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Source    string `json:"source"`
}

// LoadAutomationFromFile reads YAML from file and returns an Automation
func LoadAutomationFromFile(path string) (*Automation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tmpl Automation
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, err
	}
	return &tmpl, nil
}
