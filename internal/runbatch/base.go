// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"errors"
	"maps"
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
// If overwrite is false and Cwd is already set, it will not change the existing Cwd.
// If cwd is an empty string or the existing working directory is set, it will not change the existing Cwd.
func (c *BaseCommand) SetCwd(cwd string, overwrite bool) {
	if !overwrite && (c.Cwd != "" || cwd == "") {
		return
	}

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
func (c *BaseCommand) ShouldRun(state RunState) ShouldRunAction {
	if c.RunsOnCondition == RunOnAlways {
		// If the command always runs, return true
		return ShouldRunActionRun
	}

	if state.Err != nil {
		if errors.Is(state.Err, ErrSkipIntentional) {
			return ShouldRunActionSkip
		}

		// If there was an error in the previous run, check if the command runs on errors
		if c.RunsOnCondition == RunOnError {
			return ShouldRunActionRun
		}

		if c.RunsOnCondition == RunOnExitCodes && slices.Contains(c.RunsOnExitCodes, state.ExitCode) {
			// If the command runs on specific exit codes and the previous run's exit code matches, return true
			return ShouldRunActionRun
		}

		return ShouldRunActionError
	}

	return ShouldRunActionRun
}
