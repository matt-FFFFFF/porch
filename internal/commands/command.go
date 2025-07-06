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

	"github.com/matt-FFFFFF/porch/internal/config/hcl"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

var (
	// ErrYamlUnmarshal is returned when a YAML command definition cannot be unmarshaled.
	ErrYamlUnmarshal = errors.New(
		"failed to decode YAML command definition, please check the syntax and structure of your YAML file",
	)
	// ErrHclConfig is returned when a HCL command definition cannot be decoded.
	ErrHclConfig = errors.New(
		"failed to decode HCL command definition, please check the syntax and structure of your HCL file",
	)
	// ErrNilParent is returned when a command is created without a parent runnable context.
	ErrNilParent = errors.New(
		"command cannot be created without a parent runnable, please provide a valid parent runnable",
	)
	// ErrPath is returned when there is an error resolving the working directory path.
	ErrPath = errors.New(
		"error resolving path",
	)
	// ErrFailedToCreateRunnable is returned when a runnable command cannot be created.
	ErrFailedToCreateRunnable = errors.New(
		"failed to create runnable command, please check the command definition and ensure all required fields are set",
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
func (d *BaseDefinition) ToBaseCommand(
	_ context.Context, parent runbatch.Runnable,
) (*runbatch.BaseCommand, error) {
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

	// Process the working directory
	workingDir, relWorkingDir, err := processWorkingDirectory(
		d.WorkingDirectory,
		parent.GetCwd(),
	)

	if err != nil {
		return nil, errors.Join(ErrPath, err)
	}

	base.Cwd = workingDir
	base.CwdRel = relWorkingDir

	return base, nil
}

// HclCommandToBaseCommand converts an HCL command block to a runbatch.BaseCommand.
func HclCommandToBaseCommand(
	_ context.Context,
	hclCommand *hcl.CommandBlock,
	parent runbatch.Runnable,
) (*runbatch.BaseCommand, error) {
	if parent == nil {
		return nil, ErrNilParent
	}

	if hclCommand.RunsOnCondition == "" {
		hclCommand.RunsOnCondition = runbatch.RunOnSuccess.String()
	}

	runsOn, err := runbatch.NewRunCondition(hclCommand.RunsOnCondition)
	if err != nil {
		return nil, errors.Join(ErrHclConfig, err)
	}

	base := &runbatch.BaseCommand{
		Label:           hclCommand.Name,
		RunsOnCondition: runsOn,
		RunsOnExitCodes: hclCommand.RunsOnExitCodes,
		Env:             hclCommand.Env,
	}

	base.SetParent(parent)

	// Process the working directory
	workingDir, relWorkingDir, err := processWorkingDirectory(
		hclCommand.WorkingDirectory,
		parent.GetCwd(),
	)

	if err != nil {
		return nil, err
	}

	base.Cwd = workingDir
	base.CwdRel = relWorkingDir

	return base, nil
}

// processWorkingDirectory processes the working directory for a command.
// It returns the absolute path and the relative working directory.
func processWorkingDirectory(
	workingDirectory, parentWorkingDirectory string,
) (string, string, error) {
	if workingDirectory == "" {
		return parentWorkingDirectory, "", nil
	}

	workingDirectory = filepath.Clean(workingDirectory)

	// If it's an absolute path, use it directly
	if filepath.IsAbs(workingDirectory) {
		return workingDirectory, "", nil
	}

	// Otherwise, resolve it relative to the parent's cwd
	joined := filepath.Join(parentWorkingDirectory, workingDirectory)

	absPath, err := filepath.Abs(joined)
	if err != nil {
		return "", "", errors.Join(ErrPath, err)
	}

	return absPath, workingDirectory, nil
}
