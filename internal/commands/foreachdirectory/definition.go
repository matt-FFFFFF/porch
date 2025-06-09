// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package foreachdirectory

import (
	"errors"
	"strings"

	"github.com/matt-FFFFFF/porch/internal/commands"
)

var (
	// ErrBothCommandsAndGroup is returned when both commands and command_group are specified.
	ErrBothCommandsAndGroup = errors.New("cannot specify both 'commands' and 'command_group'")
	// ErrEmptyCommandGroup is returned when command_group is specified but is empty or whitespace.
	ErrEmptyCommandGroup = errors.New("command_group cannot be empty or whitespace")
)

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
	Commands []any `yaml:"commands,omitempty" docdesc:"List of commands to execute in each directory"`
	// CommandGroup is a reference to a named command group
	CommandGroup string `yaml:"command_group,omitempty" docdesc:"Reference to a named command group"`
}

// Validate ensures that commands and command_group are not both specified,
// and that command_group is not empty or whitespace if specified.
func (d *Definition) Validate() error {
	hasCommands := len(d.Commands) > 0
	hasCommandGroup := d.CommandGroup != ""

	if hasCommands && hasCommandGroup {
		return ErrBothCommandsAndGroup
	}

	if hasCommandGroup && strings.TrimSpace(d.CommandGroup) == "" {
		return ErrEmptyCommandGroup
	}

	return nil
}
