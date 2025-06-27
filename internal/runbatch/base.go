// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"errors"
	"maps"
	"path/filepath"
	"slices"
)

// BaseCommand is a struct that implements the Runnable interface.
// It should be embedded in other command types to provide common functionality.
type BaseCommand struct {
	Label           string            // Optional label for the command
	Cwd             string            // The working directory for the command
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
func NewBaseCommand(label, cwd string, runsOn RunCondition, runOnExitCodes []int, env map[string]string) *BaseCommand {
	if runOnExitCodes == nil {
		runOnExitCodes = []int{0} // Default to running on success (exit code 0)
	}

	if env == nil {
		env = make(map[string]string)
	}

	return &BaseCommand{
		Label:           label,
		Cwd:             cwd,
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

// SetCwd sets the working directory for the command.
// If overwrite is false and Cwd is already set to an absolute path, it will not change the existing Cwd.
// If cwd is an empty string, it will not change the existing Cwd.
// Relative paths will be resolved against the new cwd only if the new cwd is different from the existing one.
func (c *BaseCommand) SetCwd(cwd string, overwrite bool) {
	if cwd == "" {
		return
	}

	// If we have an existing cwd and it's absolute, only overwrite if explicitly requested
	if !overwrite && c.Cwd != "" && filepath.IsAbs(c.Cwd) {
		return
	}

	// If the existing cwd is relative (or empty), resolve it against the new cwd
	if c.Cwd == "" || !filepath.IsAbs(c.Cwd) {
		if c.Cwd == "" {
			c.Cwd = cwd
			return
		} else {
			// Only join paths if the existing cwd is not the same as the new cwd
			// This prevents duplicate path issues when the same relative path is set multiple times
			if c.Cwd != cwd {
				c.Cwd = filepath.Join(cwd, c.Cwd)
			}
		}

		return
	}

	// If we get here, existing cwd is absolute and overwrite is true
	c.Cwd = cwd
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
