// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package foreachdirectory

import "github.com/matt-FFFFFF/porch/internal/commands"

type definition struct {
	commands.BaseDefinition `yaml:",inline"`
	// Mode can be "parallel" or "serial"
	Mode string `yaml:"mode"`
	// Depth specifies how deep to traverse directories
	Depth int `yaml:"depth"`
	// IncludeHidden specifies whether to include hidden directories
	IncludeHidden bool `yaml:"include_hidden"`
	// WorkingDirectoryStrategy can be "none", "item_relative", or "item_absolute"
	WorkingDirectoryStrategy string `yaml:"working_directory_strategy"`
	// Commands is a list of commands to run in each directory
	Commands []any `yaml:"commands"`
}
