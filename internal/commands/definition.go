// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package commands

// BaseDefinition contains fields common to all command types.
type BaseDefinition struct {
	// Type is the type of command, e.g., "shell", "serial", "parallel", etc.
	Type string `yaml:"type" docdesc:"The type of command (e.g., 'shell', 'serial', 'parallel', 'foreachdirectory')"` //nolint:lll
	// Name is the descriptive name of the command.
	Name string `yaml:"name" docdesc:"Descriptive name for the command"` //nolint:lll
	// WorkingDirectory is the directory in which the command should be run.
	WorkingDirectory string `yaml:"working_directory,omitempty" docdesc:"Directory in which the command should be executed"` //nolint:lll
	// RunsOnCondition can be success, error, always, or exit-codes.
	RunsOnCondition string `yaml:"runs_on_condition,omitempty" docdesc:"Condition that determines when this command runs: 'success', 'error', 'always', or 'exit-codes'"` //nolint:lll
	// RunsOnExitCodes is used when RunsOn is set to exit-codes.
	RunsOnExitCodes []int `yaml:"runs_on_exit_codes,omitempty" docdesc:"Specific exit codes that trigger execution (used with runs_on_condition: exit-codes)"` //nolint:lll
	// Env is a map of environment variables to be set for the command.
	Env map[string]string `yaml:"env,omitempty" docdesc:"Environment variables to set for the command"` //nolint:lll
}
