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
