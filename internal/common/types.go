package common

type Done struct{}

type Resource struct {
	ApiVersion string   `json:"apiVersion" yaml:"apiVersion"`
	Type       string   `json:"type" yaml:"type"`
	Metadata   Metadata `json:"metadata" yaml:"metadata"`
}

type Metadata struct {
	Name        string            `json:"name" yaml:"name"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type AutomationLog struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

type ServiceLog struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}
