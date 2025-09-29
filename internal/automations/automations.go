package automations

import "time"

type AutomationSpec struct {
	// VolumeMounts defines any volume mounts in play when containers are
	// spun up
	VolumeMounts []VolumeMount `json:"volumeMounts" yaml:"volumeMounts"`

	// Phases defines the various steps of the automation
	Phases []Phase `json:"phases" yaml:"phases"`

	// Variables is used during processing but not during definition
	Variables VariablesSpec `json:"variables" yaml:"-"`

	// Status is used during processing but not during definition
	Status AutomationStatus `json:"status" yaml:"-"`
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

type AutomationStatus struct {
	Id       string    `json:"id"`
	QueuedAt time.Time `json:"queuedAt"`
}
