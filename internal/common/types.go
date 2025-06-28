package common

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
