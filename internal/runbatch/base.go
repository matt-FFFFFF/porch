// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
)

var (
	// ErrSetCwd is returned when setting the working directory fails.
	ErrSetCwd = errors.New("failed to set working directory, please check the path and permissions")
	// ErrPathNotAbsolute is returned when a command's working directory is not absolute.
	ErrPathNotAbsolute = errors.New("path must be absolute, all commands must have absolute cwd")
)

// BaseCommand is a struct that implements the Runnable interface.
// It should be embedded in other command types to provide common functionality.
type BaseCommand struct {
	Label           string            // Optional label for the command
	Cwd             string            // The absolute working directory for the command
	CwdRel          string            // The relative working directory for the command, from the source YAML file
	CwdIsTemp       bool              // Indicates if the cwd is a temporary directory
	RunsOnCondition RunCondition      // The condition under which the command runs
	RunsOnExitCodes []int             // Specific exit codes that trigger the command to run
	Env             map[string]string // Environment variables to be passed to the command
	parent          Runnable          // The parent command or batch, if any
}

// PreviousCommandStatus holds the state of the previous command execution.
type PreviousCommandStatus struct {
	// State is the result status of the previous command.
	State ResultStatus
	// ExitCode is the exit code of the previous command.
	ExitCode int
	// Err is the error from the previous command, if any.
	Err error
}

// NewBaseCommand creates a new BaseCommand with the specified parameters.
func NewBaseCommand(
	label, cwd, relPath string, runsOn RunCondition, runOnExitCodes []int, env map[string]string,
) *BaseCommand {
	if runOnExitCodes == nil {
		runOnExitCodes = []int{0} // Default to running on success (exit code 0)
	}

	if env == nil {
		env = make(map[string]string)
	}

	return &BaseCommand{
		Label:           label,
		Cwd:             cwd,
		CwdRel:          relPath,
		RunsOnCondition: runsOn,
		RunsOnExitCodes: runOnExitCodes,
		Env:             env,
	}
}

// GetLabel returns the label of the command.
func (c *BaseCommand) GetLabel() string {
	if c.Label == "" {
		return "Command"
	}

	return c.Label
}

// GetParent returns the parent for this command or batch.
func (c *BaseCommand) GetParent() Runnable {
	return c.parent
}

// SetParent sets the parent for this command or batch.
func (c *BaseCommand) SetParent(parent Runnable) {
	c.parent = parent
}

// GetCwd returns the current working directory for the command.
func (c *BaseCommand) GetCwd() string {
	return c.Cwd
}

// GetCwdRel returns the relative working directory for the command, from the source YAML file.
func (c *BaseCommand) GetCwdRel() string {
	return c.CwdRel
}

// SetCwd sets the working directory for the command.
// All commands MUST have an absolute cwd at all times.
// This method requires the current cwd to be absolute and will error otherwise.
func (c *BaseCommand) SetCwdToSpecificAbsolute(cwd string) error {
	if cwd == "" {
		return nil
	}

	if !filepath.IsAbs(cwd) {
		return fmt.Errorf(
			"%w: new working directory %q is not absolute, all commands must have absolute cwd", ErrPathNotAbsolute, cwd,
		)
	}

	// Current working directory must be absolute
	if !filepath.IsAbs(c.Cwd) {
		return fmt.Errorf(
			"%w: current working directory %q is not absolute, all commands must have absolute cwd", ErrSetCwd, c.Cwd,
		)
	}

	c.Cwd = cwd

	return nil
}

// SetCwd sets the working directory for the command.
// All commands MUST have an absolute cwd at all times.
// This method requires the current cwd to be absolute and will error otherwise.
func (c *BaseCommand) SetCwd(cwd string) error {
	if cwd == "" {
		return nil
	}

	// Current working directory must be absolute
	if !filepath.IsAbs(c.Cwd) {
		return fmt.Errorf(
			"%w: current working directory %q is not absolute, all commands must have absolute cwd", ErrSetCwd, c.Cwd,
		)
	}

	if filepath.IsAbs(cwd) {
		// If the new cwd is absolute, we can set it directly
		// using the relative path if it exists.
		parent := c.GetParent()
		if parent == nil {
			return fmt.Errorf("%w: parent command is not set, cannot determine relative working directory", ErrSetCwd)
		}

		c.Cwd = filepath.Join(cwd, parent.GetCwdRel(), c.CwdRel)

		return nil
	}

	// If the new cwd is relative, resolve it against the current absolute cwd
	c.Cwd = filepath.Join(c.Cwd, cwd)

	return nil
}

// InheritEnv sets additional environment variables for the command.
func (c *BaseCommand) InheritEnv(env map[string]string) {
	if len(c.Env) == 0 {
		c.Env = maps.Clone(env)
		return
	}

	for k, v := range maps.All(env) {
		if _, ok := c.Env[k]; !ok {
			c.Env[k] = v
		}
	}
}

// ShouldRun checks if the command should run based on the current state.
// It returns a ShouldRunAction indicating whether to run, skip, or error.
func (c *BaseCommand) ShouldRun(prev PreviousCommandStatus) ShouldRunAction {
	switch c.RunsOnCondition {
	case RunOnAlways:
		return ShouldRunActionRun
	case RunOnSuccess:
		if prev.State != ResultStatusSuccess {
			return ShouldRunActionError
		}

		if errors.Is(prev.Err, ErrSkipIntentional) {
			return ShouldRunActionSkip
		}

		return ShouldRunActionRun
	case RunOnExitCodes:
		if !slices.Contains(c.RunsOnExitCodes, prev.ExitCode) {
			return ShouldRunActionSkip
		}

		return ShouldRunActionRun
	case RunOnError:
		if prev.State != ResultStatusError {
			return ShouldRunActionError
		}

		return ShouldRunActionRun
	}

	return ShouldRunActionRun
}

// Run does nothing, but is required to implement the Runnable interface.
// It should be overridden by concrete command types to provide actual functionality.
func (c *BaseCommand) Run(_ context.Context) Results {
	return nil
}
