// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package commands

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

var (
	// ErrYamlUnmarshal is returned when a YAML command definition cannot be unmarshaled.
	ErrYamlUnmarshal = errors.New(
		"failed to decode YAML command definition, please check the syntax and structure of your YAML file",
	)
	// ErrNilParent is returned when a command is created without a parent runnable context.
	ErrNilParent = errors.New(
		"command cannot be created without a parent runnable, please provide a valid parent runnable",
	)
	// ErrPath is returned when there is an error resolving the working directory path.
	ErrPath = errors.New(
		"error resolving path",
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
// It resolves the working directory against the parent's cwd if available in context.
func (d *BaseDefinition) ToBaseCommand(ctx context.Context, parent runbatch.Runnable) (*runbatch.BaseCommand, error) {
	if d.RunsOnCondition == "" {
		d.RunsOnCondition = runbatch.RunOnSuccess.String()
	}

	if parent == nil {
		return nil, ErrNilParent
	}

	ro, err := runbatch.NewRunCondition(d.RunsOnCondition)
	if err != nil {
		return nil, errors.Join(ErrYamlUnmarshal, err)
	}

	base := &runbatch.BaseCommand{
		Label:           d.Name,
		RunsOnCondition: ro,
		RunsOnExitCodes: slices.Clone(d.RunsOnExitCodes),
		Env:             maps.Clone(d.Env),
	}

	base.SetParent(parent)

	// Make a copy to avoid side effects
	defWd := filepath.Clean(d.WorkingDirectory)

	// Resolve the working directory against parent cwd if available
	if defWd == "" {
		// If no working directory is specified, use the parent's cwd
		base.Cwd = parent.GetCwd()
		return base, nil
	}

	// If it's an absolute path, use it directly
	if filepath.IsAbs(defWd) {
		base.Cwd = defWd
		return base, nil
	}

	base.CwdRel = defWd

	// Otherwise, resolve it relative to the parent's cwd
	joined := filepath.Join(parent.GetCwd(), defWd)
	joined, err = filepath.Abs(joined)
	if err != nil {
		return nil, errors.Join(ErrPath, err)
	}

	base.Cwd = joined

	return base, nil
}
