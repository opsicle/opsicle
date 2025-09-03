package automations

import (
	"opsicle/internal/approvals"
	"opsicle/internal/common"
	"os"

	"gopkg.in/yaml.v3"
)

type Template struct {
	common.Resource `json:"resource" yaml:",inline"`
	Spec            Spec `json:"spec" yaml:"spec"`
}

func (t Template) GetName() string {
	return t.Resource.Metadata.Name
}

func (t Template) GetDescription() string {
	description, ok := t.Resource.Metadata.Labels["opsicle.io/description"]
	if !ok {
		return ""
	}
	return description
}

type Spec struct {
	// ApprovalPolicy defines the approval mechanism
	ApprovalPolicy *ApprovalPolicySpec `json:"approvalPolicy" yaml:"approvalPolicy"`

	// Metadata defines other metadata not included in the parent
	// resource
	Metadata MetadataSpec `json:"metadata" yaml:"metadata"`

	// Template defines an Automation specification
	Template AutomationSpec `json:"template" yaml:"template"`
}

// ApprovalPolicySpec defines the approval mechanism in play for
// the automation
type ApprovalPolicySpec struct {
	// PolicyRef when defined should be a string that references an
	// existing policy which can be retrieved by the controller
	PolicyRef *string `json:"policyRef" yaml:"policyRef"`

	// Spec contains an inline approval policy
	Spec *approvals.PolicySpec `json:"spec" yaml:"spec"`
}

type MetadataSpec struct {
	Description string     `json:"description" yaml:"description"`
	DisplayName string     `json:"displayName" yaml:"displayName"`
	Owners      []OwnerRef `json:"owners" yaml:"owners"`
}

type OwnerRef struct {
	Name  string `json:"name" yaml:"name"`
	Email string `json:"email" yaml:"email"`
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
