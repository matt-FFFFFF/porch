// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package commands

import (
	"errors"
	"fmt"
	"maps"
	"slices"

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

// ToBaseCommand converts the BaseDefinition to a runbatch.BaseCommand.
func (d *BaseDefinition) ToBaseCommand() (*runbatch.BaseCommand, error) {
	if d.RunsOnCondition == "" {
		d.RunsOnCondition = runbatch.RunOnSuccess.String()
	}

	ro, err := runbatch.NewRunCondition(d.RunsOnCondition)
	if err != nil {
		return nil, errors.Join(ErrYamlUnmarshal, err)
	}

	return &runbatch.BaseCommand{
		Label:           d.Name,
		Cwd:             d.WorkingDirectory,
		RunsOnCondition: ro,
		RunsOnExitCodes: slices.Clone(d.RunsOnExitCodes),
		Env:             maps.Clone(d.Env),
	}, nil
}
