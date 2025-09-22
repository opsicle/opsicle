package automations

import (
	"os"

	"gopkg.in/yaml.v3"
)

type TemplateSpec struct {
	// ApprovalPolicy defines the approval mechanism
	ApprovalPolicy *ApprovalPolicySpec `json:"approvalPolicy" yaml:"approvalPolicy"`

	// Metadata defines other metadata not included in the parent
	// resource
	Metadata MetadataSpec `json:"metadata" yaml:"metadata"`

	// Variables defines the variables that should be input by a user
	// before the automation can be executed
	Variables VariablesSpec `json:"variables" yaml:"variables"`

	// Template defines an Automation specification
	Template AutomationSpec `json:"template" yaml:"template"`
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
