// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package foreachdirectory

import "github.com/matt-FFFFFF/porch/internal/commands"

// Definition represents the YAML configuration for the foreachdirectory command.
type Definition struct {
	commands.BaseDefinition `yaml:",inline"`
	// Mode can be "parallel" or "serial"
	Mode string `yaml:"mode" docdesc:"Execution mode: 'parallel' or 'serial'"`
	// Depth specifies how deep to traverse directories
	Depth int `yaml:"depth" docdesc:"Directory traversal depth (0 for unlimited)"`
	// IncludeHidden specifies whether to include hidden directories
	IncludeHidden bool `yaml:"include_hidden" docdesc:"Whether to include hidden directories in traversal"`
	// WorkingDirectoryStrategy can be "none", "item_relative", or "item_absolute"
	WorkingDirectoryStrategy string `yaml:"working_directory_strategy" docdesc:"Strategy for setting working directory: 'none', 'item_relative', or 'item_absolute'"`
	// Commands is a list of commands to run in each directory
	Commands []any `yaml:"commands" docdesc:"List of commands to execute in each directory"`
}
