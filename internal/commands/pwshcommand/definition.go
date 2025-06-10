// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package pwshcommand

import (
	"github.com/matt-FFFFFF/porch/internal/commands"
)

// Definition represents the YAML configuration for the serial command.
type Definition struct {
	commands.BaseDefinition `yaml:",inline"`
	// The command to execute, can be a path or a command name.
	ScriptFile string `yaml:"script_file" docdesc:"The path to the .ps1 file to execute"` //nolint:lll
	// Script is the PowerShell script to execute, can be a string or a file path.
	Script string `yaml:"script,omitempty" docdesc:"The PowerShell script to execute, defined in-line"` //nolint:lll
	// Exit codes that indicate success, defaults to 0.
	SuccessExitCodes []int `yaml:"success_exit_codes,omitempty" docdesc:"Exit codes that indicate success, defaults to 0"` //nolint:lll
	// Exit codes that indicate skip remaining tasks, defaults to empty.
	SkipExitCodes []int `yaml:"skip_exit_codes,omitempty" docdesc:"Exit codes that indicate skip remaining tasks, defaults to empty"` //nolint:lll
}
