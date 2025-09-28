package automations

type VariablesSpec []VariableSpec

type VariableSpec struct {
	Default     any    `json:"default" yaml:"default"`
	Description string `json:"description" yaml:"description"`
	Id          string `json:"id" yaml:"id"`
	Label       string `json:"label" yaml:"label"`
	Type        string `json:"type" yaml:"type"`
	IsRequired  bool   `json:"isRequired" yaml:"isRequired"`

	// Value is not in the spec but used during processing
	Value any `json:"value" yaml:"-"`
}
