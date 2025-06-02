// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
)

// Runnable is an interface for something that can be run as part of a batch (either a Command or a nested Batch).
type Runnable interface {
	// Run executes the command or batch and returns the results.
	// It should handle context cancellation and passing signals to any spawned process.
	Run(context.Context) Results
	// SetCwd sets the working directory for the command or batch.
	// It should be called before Run() to ensure the command or batch runs in the correct directory.
	// The boolean parameter indicates whether to overwrite the current working directory if it is set.
	SetCwd(string, bool)
	// InheritEnv sets the environment variables for the command or batch.
	// It should not overwrite the existing environment variables, but rather add to them.
	InheritEnv(map[string]string)
	// GetLabel returns the label or description of the command or batch.
	GetLabel() string
	// GetParent returns the parent for this command or batch.
	GetParent() Runnable
	// SetParent sets the parent for this command or batch.
	SetParent(Runnable)
	// ShouldRun returns true if the command or batch should be run.
	ShouldRun(state RunState) bool
}

// RunState represents the state of the previous run of a command or batch.
type RunState struct {
	ExitCode int   // Exit code of the last run
	Err      error // Error of the last run
}
