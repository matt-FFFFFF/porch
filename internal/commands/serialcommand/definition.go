// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package serialcommand

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

// Definition represents the YAML configuration for the serial command.
type Definition struct {
	commands.BaseDefinition `yaml:",inline"`
	Commands                []any  `yaml:"commands,omitempty" docdesc:"List of commands to execute sequentially"`
	CommandGroup            string `yaml:"command_group,omitempty" docdesc:"Reference to a named command group"`
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
