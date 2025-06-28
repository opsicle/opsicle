package automationtemplates

import (
	"opsicle/internal/common"
	"os"

	"gopkg.in/yaml.v3"
)

type AutomationTemplate struct {
	common.Resource `json:"resource" yaml:",inline"`
	Spec            Spec `json:"spec" yaml:"spec"`
}

type Spec struct {
	Metadata SpecMetadata `json:"metadata" yaml:"metadata"`
	Template SpecTemplate `json:"template" yaml:"template"`
}

type SpecMetadata struct {
	DisplayName string     `json:"displayName" yaml:"displayName"`
	Owners      []OwnerRef `json:"owners" yaml:"owners"`
}

type OwnerRef struct {
	Name  string `json:"name" yaml:"name"`
	Email string `json:"email" yaml:"email"`
}

type SpecTemplate struct {
	VolumeMounts []VolumeMount `json:"volumeMounts" yaml:"volumeMounts"`
	Phases       []Phase       `json:"phases" yaml:"phases"`
}

type VolumeMount struct {
	Host      string `json:"host" yaml:"host"`
	Container string `json:"container" yaml:"container"`
}

type Phase struct {
	Name     string   `json:"name" yaml:"name"`
	Image    string   `json:"image" yaml:"image"`
	Commands []string `json:"command" yaml:"commands"`
	Timeout  int      `json:"timeout" yaml:"timeout"`
}

// ToYaml converts AutomationTemplate to YAML
func (a *AutomationTemplate) ToYaml() ([]byte, error) {
	return yaml.Marshal(a)
}

// ToFile writes the serialized YAML to a file
func (a *AutomationTemplate) ToFile(path string) error {
	data, err := a.ToYaml()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadFromFile reads YAML from file and returns an AutomationTemplate
func LoadFromFile(path string) (*AutomationTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tmpl AutomationTemplate
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, err
	}
	return &tmpl, nil
}
