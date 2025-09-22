package automations

import (
	"os"

	"gopkg.in/yaml.v3"
)

func LoadAutomation(data []byte) (*Automation, error) {
	var automation Automation
	if err := yaml.Unmarshal(data, &automation); err != nil {
		return nil, err
	}
	return &automation, nil
}

// LoadAutomationFromFile reads YAML from file and returns an Automation
func LoadAutomationFromFile(path string) (*Automation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadAutomation(data)
}

func LoadAutomationTemplate(data []byte) (*Template, error) {
	var template Template
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, err
	}
	return &template, nil
}

// LoadAutomationTemplateFromFile reads YAML from file and returns a Template
func LoadAutomationTemplateFromFile(path string) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadAutomationTemplate(data)
}
