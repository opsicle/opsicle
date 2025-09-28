package automations

import "opsicle/internal/common"

type Automation struct {
	common.Resource `json:"resource" yaml:",inline"`
	Spec            AutomationSpec `json:"spec" yaml:"spec"`
}

type Template struct {
	common.Resource `json:"resource" yaml:",inline"`
	Spec            TemplateSpec `json:"spec" yaml:"spec"`
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

// GetVariables returns a mapping of variable ID to a
// VariableSpec; returns nil if there are no variables
func (t Template) GetVariables() map[string]VariableSpec {
	if t.Spec.Variables != nil {
		varMap := map[string]VariableSpec{}
		for _, variable := range t.Spec.Variables {
			varMap[variable.Id] = variable
		}
		if len(varMap) == 0 {
			return nil
		}
		return varMap
	}
	return nil
}
