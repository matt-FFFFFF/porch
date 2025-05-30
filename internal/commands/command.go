package commands

// BaseDefinition contains fields common to all command types
type BaseDefinition struct {
	Type string `yaml:"type"`
	Name string `yaml:"name"`
	Cwd  string `yaml:"cwd,omitempty"`
}
