package rillv1beta

type Source struct {
	Type   string
	URI    string `yaml:"uri,omitempty"`
	Path   string `yaml:"path,omitempty"`
	Region string `yaml:"region,omitempty"`
}

type ProjectConfig struct {
	// Project variables
	Variables   map[string]string `yaml:"env,omitempty"`
	Title       string            `yaml:"title,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Access      Access            `yaml:"access,omitempty"`
}

// Access TODO add validations
type Access struct {
	Claims             []Claims `yaml:"claims,omitempty"`
	DefaultModelAccess string   `yaml:"default_model_access,omitempty"`
}

type Claims struct {
	Name    string   `yaml:"name,omitempty"`
	Type    string   `yaml:"type,omitempty"`
	Default string   `yaml:"default,omitempty"`
	Options []string `yaml:"options,omitempty"`
}
