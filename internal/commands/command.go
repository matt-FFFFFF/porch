// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package commands

import (
	"errors"
	"fmt"

	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

var (
	// ErrYamlUnmarshal is returned when a YAML command definition cannot be unmarshaled.
	ErrYamlUnmarshal = errors.New(
		"failed to decode YAML command definition, please check the syntax and structure of your YAML file",
	)
)

// ErrCommandCreate is returned when a command cannot be created.
// It includes the command name for easier debugging.
type ErrCommandCreate struct {
	cmdName string
}

// Error implements the error interface for ErrCommandCreate.
func (e *ErrCommandCreate) Error() string {
	return fmt.Sprintf("failed to create command %q", e.cmdName)
}

// NewErrCommandCreate creates a new ErrCommandCreate error.
func NewErrCommandCreate(cmdName string) error {
	return &ErrCommandCreate{cmdName: cmdName}
}

// BaseDefinition contains fields common to all command types.
type BaseDefinition struct {
	// Type is the type of command, e.g., "shell", "serial", "parallel", etc.
	Type string `yaml:"type"`
	// Name is the descriptive name of the command.
	Name string `yaml:"name"`
	// WorkingDirectory is the directory in which the command should be run.
	WorkingDirectory string `yaml:"working_directory,omitempty"`
	// RunsOnCondition can be success, error, always, or exit-codes.
	RunsOnCondition string `yaml:"runs_on_condition,omitempty"`
	// RunsOnExitCodes is used when RunsOn is set to exit-codes.
	RunsOnExitCodes []int `yaml:"runs_on_exit_codes,omitempty"`
	// Env is a map of environment variables to be set for the command.
	Env map[string]string `yaml:"env,omitempty"`
}

// ToBaseCommand converts the BaseDefinition to a runbatch.BaseCommand.
func (d *BaseDefinition) ToBaseCommand() (*runbatch.BaseCommand, error) {
	ro, err := runbatch.NewRunCondition(d.RunsOnCondition)
	if err != nil {
		return nil, errors.Join(ErrYamlUnmarshal, err)
	}

	return &runbatch.BaseCommand{
		Label:           d.Name,
		Cwd:             d.WorkingDirectory,
		RunsOnCondition: ro,
		RunsOnExitCodes: d.RunsOnExitCodes,
		Env:             d.Env,
	}, nil
}
