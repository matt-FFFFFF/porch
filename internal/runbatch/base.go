// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"errors"
	"maps"
	"path/filepath"
	"slices"
	"sync"

	"github.com/matt-FFFFFF/porch/internal/progress"
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
	RunsOnCondition RunCondition      // The condition under which the command runs
	RunsOnExitCodes []int             // Specific exit codes that trigger the command to run
	Env             map[string]string // Environment variables to be passed to the command
	parent          Runnable          // The parent command or batch, if any
	cwd             string            // The working directory for the command, can be absolute or relative. Do not access directly, use GetCwd() and SetCwd() instead.
	reporterMu      sync.RWMutex      // Mutex to protect the reporter field
	reporter        progress.Reporter // Optional progress reporter for real-time updates
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
	label, cwd string, runsOn RunCondition, runOnExitCodes []int, env map[string]string,
) *BaseCommand {
	if runOnExitCodes == nil {
		runOnExitCodes = []int{0} // Default to running on success (exit code 0)
	}

	if env == nil {
		env = make(map[string]string)
	}

	return &BaseCommand{
		Label:           label,
		cwd:             cwd,
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
// It resolves the working directory using the following rules:
//   - If the receiver is nil, returns "."
//   - If cwd is empty and no parent exists, returns "."
//   - If cwd is empty and parent exists, inherits parent's cwd
//   - If cwd is absolute, returns it directly
//   - If cwd is relative and no parent exists, returns the relative path
//   - If cwd is relative and parent exists, joins it with parent's cwd
func (c *BaseCommand) GetCwd() string {
	if c == nil {
		return "."
	}

	if c.cwd == "" {
		if c.parent == nil {
			return "."
		}

		return c.parent.GetCwd()
	}

	if filepath.IsAbs(c.cwd) {
		return c.cwd
	}

	if c.parent == nil {
		return c.cwd
	}

	return filepath.Join(c.parent.GetCwd(), c.cwd)
}

// SetCwd sets the working directory for the command.
func (c *BaseCommand) SetCwd(cwd string) error {
	c.cwd = cwd
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

// SetProgressReporter sets an optional progress reporter for real-time execution updates.
// If not set (nil), the command will run without progress reporting.
// This method is thread-safe but should be called before Run() for proper behavior.
func (c *BaseCommand) SetProgressReporter(reporter progress.Reporter) {
	c.reporterMu.Lock()
	defer c.reporterMu.Unlock()

	c.reporter = reporter
}

// GetProgressReporter returns the currently set progress reporter, or nil if none is set.
// This method is thread-safe.
func (c *BaseCommand) GetProgressReporter() progress.Reporter {
	c.reporterMu.RLock()
	defer c.reporterMu.RUnlock()

	return c.reporter
}

// hasProgressReporter returns true if a progress reporter has been set.
func (c *BaseCommand) hasProgressReporter() bool {
	c.reporterMu.RLock()
	defer c.reporterMu.RUnlock()

	return c.reporter != nil
}

// GetType returns the type of the runnable (e.g., "Command", "SerialBatch", "ParallelBatch", etc.).
func (c *BaseCommand) GetType() string {
	return "BaseCommand"
}
