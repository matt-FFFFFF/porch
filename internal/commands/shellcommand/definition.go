// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package shellcommand

import "github.com/matt-FFFFFF/porch/internal/commands"

// Definition is the YAML definition for the shell command.
type Definition struct {
	commands.BaseDefinition `yaml:",inline"`
	// The command to execute, can be a path or a command name.
	CommandLine string `yaml:"command_line" docdesc:"The command to execute, can be a path or a command name"` //nolint:lll
	// Exit codes that indicate success, defaults to 0.
	SuccessExitCodes []int `yaml:"success_exit_codes,omitempty" docdesc:"Exit codes that indicate success, defaults to 0"` //nolint:lll
	// Exit codes that indicate skip remaining tasks, defaults to empty.
	SkipExitCodes []int `yaml:"skip_exit_codes,omitempty" docdesc:"Exit codes that indicate skip remaining tasks, defaults to empty"` //nolint:lll
}
