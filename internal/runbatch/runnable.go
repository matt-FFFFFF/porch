// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"

	"github.com/matt-FFFFFF/porch/internal/progress"
)

// Runnable is an interface for something that can be run as part of a batch (either a Command or a nested Batch).
type Runnable interface {
	// Run executes the command or batch and returns the results.
	// It should handle context cancellation and passing signals to any spawned process.
	Run(context.Context) Results
	// GetCwd returns the current working directory for the command or batch.
	GetCwd() string
	// SetCwd sets the working directory for the command or batch.
	// It should be called before Run() to ensure the command or batch runs in the correct directory.
	SetCwd(string) error
	// SetCwdToSpecificAbsolute sets the working directory to a specific absolute path.
	SetCwdToSpecificAbsolute(string) error
	// GetCwdRel returns the relative working directory for the command or batch, from the source YAML file.
	GetCwdRel() string
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
	ShouldRun(state PreviousCommandStatus) ShouldRunAction
	// SetProgressReporter sets an optional progress reporter for real-time execution updates.
	// If not set (nil), the command will run without progress reporting.
	// This method must be called before Run() and must not be called concurrently with Run() or GetProgressReporter().
	SetProgressReporter(reporter progress.Reporter)
	// GetProgressReporter returns the currently set progress reporter, or nil if none is set.
	GetProgressReporter() progress.Reporter
}

// RunnableWithChildren is an interface for runnables that can have child commands or batches.
type RunnableWithChildren interface {
	// GetChildren returns the child commands or batches of this runnable.
	GetChildren() []Runnable
}
