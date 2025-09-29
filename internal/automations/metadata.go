package automations

type MetadataSpec struct {
	Description string     `json:"description" yaml:"description"`
	DisplayName string     `json:"displayName" yaml:"displayName"`
	Owners      []OwnerRef `json:"owners" yaml:"owners"`
}

type OwnerRef struct {
	Name  string `json:"name" yaml:"name"`
	Email string `json:"email" yaml:"email"`
}
