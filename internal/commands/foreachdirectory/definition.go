// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package foreachdirectory

import "github.com/matt-FFFFFF/porch/internal/commands"

type definition struct {
	commands.BaseDefinition  `yaml:",inline"`
	Mode                     string `yaml:"mode"`                       // Mode can be "parallel" or "serial"
	Depth                    int    `yaml:"depth"`                      // Depth specifies how deep to traverse directories
	IncludeHidden            bool   `yaml:"include_hidden"`             // IncludeHidden specifies whether to include hidden directories
	WorkingDirectoryStrategy string `yaml:"working_directory_strategy"` // WorkingDirectoryStrategy can be "none", "item_relative", or "item_absolute"
	Commands                 []any  `yaml:"commands"`                   // Commands is a list of commands to run in each directory
}
