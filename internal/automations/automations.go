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
