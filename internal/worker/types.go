package worker

import (
	"opsicle/internal/automations"
	"opsicle/internal/common"
)

type automationSpec struct {
	Phases       []automations.Phase       `json:"phases" yaml:"phases"`
	VolumeMounts []automations.VolumeMount `json:"volumeMounts" yaml:"volumeMounts"`

	AutomationLogs chan string            `json:"-"`
	ServiceLogs    chan common.ServiceLog `json:"-"`
}
