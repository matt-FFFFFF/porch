package foreachdirectory

import "github.com/matt-FFFFFF/porch/internal/commands"

type definition struct {
	commands.BaseDefinition `yaml:",inline"`
	Mode                    string `yaml:"mode"`           // Mode can be "parallel" or "serial"
	Depth                   int    `yaml:"depth"`          // Depth specifies how deep to traverse directories
	IncludeHidden           bool   `yaml:"include_hidden"` // IncludeHidden specifies whether to include hidden directories
	Commands                []any  `yaml:"commands"`
}
