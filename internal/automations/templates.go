package automations

import (
	"opsicle/internal/common"
	"os"

	"gopkg.in/yaml.v3"
)

type Template struct {
	common.Resource `json:"resource" yaml:",inline"`
	Spec            Spec `json:"spec" yaml:"spec"`
}

type Spec struct {
	Metadata SpecMetadata `json:"metadata" yaml:"metadata"`
	// Approval ApprovalSpec   `json:"approval" yaml:"approval"`
	Template AutomationSpec `json:"template" yaml:"template"`
}

type SpecMetadata struct {
	DisplayName string     `json:"displayName" yaml:"displayName"`
	Owners      []OwnerRef `json:"owners" yaml:"owners"`
}

type OwnerRef struct {
	Name  string `json:"name" yaml:"name"`
	Email string `json:"email" yaml:"email"`
}

type AutomationSpec struct {
	VolumeMounts []VolumeMount `json:"volumeMounts" yaml:"volumeMounts"`
	Phases       []Phase       `json:"phases" yaml:"phases"`
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
	Logs     PhaseLogs `json:"logs" yaml:"logs"`
}

type PhaseLogs []PhaseLog

type PhaseLog struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Source    string `json:"source"`
}

// ToYaml converts Template to YAML
func (a *Template) ToYaml() ([]byte, error) {
	return yaml.Marshal(a)
}

// ToFile writes the serialized YAML to a file
func (a *Template) ToFile(path string) error {
	data, err := a.ToYaml()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadAutomationTemplateFromFile reads YAML from file and returns a Template
func LoadAutomationTemplateFromFile(path string) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tmpl Template
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, err
	}
	return &tmpl, nil
}
