// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"time"

	"github.com/matt-FFFFFF/porch/internal/progress"
)

// Ensure ForEachCommand implements ProgressiveRunnable.
var _ ProgressiveRunnable = (*ForEachCommand)(nil)

// RunWithProgress implements ProgressiveRunnable for ForEachCommand.
func (f *ForEachCommand) RunWithProgress(ctx context.Context, reporter progress.ProgressReporter) Results {
	// Our command path is just our label - the reporter will handle prefixing
	commandPath := []string{f.Label}

	// Report that this foreach is starting
	reporter.Report(progress.ProgressEvent{
		CommandPath: commandPath,
		Type:        progress.EventStarted,
		Message:     "Starting foreach command",
		Timestamp:   time.Now(),
	})

	// Use the original Run method but with progressive reporting
	// We'll override the child commands to be progressive if possible
	results := f.runWithProgressiveChildren(ctx, reporter)

	// Report completion based on results
	if results.HasError() {
		reporter.Report(progress.ProgressEvent{
			CommandPath: commandPath,
			Type:        progress.EventFailed,
			Message:     "ForEach command failed",
			Timestamp:   time.Now(),
			Data: progress.EventData{
				ExitCode: -1,
				Error:    ErrResultChildrenHasError,
			},
		})
	} else {
		reporter.Report(progress.ProgressEvent{
			CommandPath: commandPath,
			Type:        progress.EventCompleted,
			Message:     "ForEach command completed successfully",
			Timestamp:   time.Now(),
			Data: progress.EventData{
				ExitCode: 0,
			},
		})
	}

	return results
}

// runWithProgressiveChildren runs the foreach command using the original logic
// but with progressive reporting for child batches.
func (f *ForEachCommand) runWithProgressiveChildren(ctx context.Context, reporter progress.ProgressReporter) Results {
	// This is mostly copied from the original Run method, but with progressive execution
	result := &Result{
		Label:    f.Label,
		ExitCode: 0,
		Children: Results{},
		Status:   ResultStatusSuccess,
	}

	// Get the items to iterate over
	items, err := f.ItemsProvider(ctx, f.Cwd)
	if err != nil {
		result.Error = err
		result.ExitCode = -1
		result.Status = ResultStatusError

		return Results{result}
	}

	// Handle empty list
	if len(items) == 0 {
		// Not an error, just an empty list - return success
		return Results{result}
	}

	// Create item batches like the original implementation
	foreachCommands := make([]Runnable, len(items))

	for i, item := range items {
		// Clone the current environment for each item
		newEnv := maps.Clone(f.Env)
		if newEnv == nil {
			newEnv = make(map[string]string)
		}

		var cwd string

		switch f.CwdStrategy {
		case CwdStrategyItemRelative:
			cwd = filepath.Join(f.Cwd, item)
		case CwdStrategyItemAbsolute:
			cwd = item
		default:
			cwd = f.Cwd // No modification to cwd
		}

		newEnv[ItemEnvVar] = item
		base := NewBaseCommand(
			fmt.Sprintf("[%s]", item),
			cwd,
			f.RunsOnCondition,
			f.RunsOnExitCodes,
			newEnv,
		)

		// Create the serial batch for this item
		serialBatch := &SerialBatch{
			BaseCommand: base,
		}

		// Clone the commands for each iteration to avoid shared state
		clonedCommands := make([]Runnable, len(f.Commands))
		for j, cmd := range f.Commands {
			clonedCommands[j] = cloneRunnable(cmd)
			// Set the parent of each cloned command to this serial batch
			clonedCommands[j].SetParent(serialBatch)
		}

		serialBatch.Commands = clonedCommands
		foreachCommands[i] = serialBatch
	}

	base := NewBaseCommand(f.Label, f.Cwd, f.RunsOnCondition, f.RunsOnExitCodes, maps.Clone(f.Env))
	base.SetParent(f.GetParent())

	// Handle different execution modes with progressive execution
	var run Runnable

	if f.Mode == ForEachParallel {
		base.Label = f.Label + " (parallel)"
		run = &ParallelBatch{
			BaseCommand: base,
			Commands:    foreachCommands,
		}
	} else {
		base.Label = f.Label + " (serial)"
		run = &SerialBatch{
			BaseCommand: base,
			Commands:    foreachCommands,
		}
	}

	// Set the parent for each foreach command to the run batch
	for _, foreachCmd := range foreachCommands {
		foreachCmd.SetParent(run)
	}

	// Execute with progress reporting
	var results Results

	if progressive, ok := run.(ProgressiveRunnable); ok {
		// Create a child reporter for the batch execution
		childReporter := NewChildReporter(reporter, []string{f.Label})
		results = progressive.RunWithProgress(ctx, childReporter)
	} else {
		// Fallback to regular execution with basic progress events
		childReporter := NewChildReporter(reporter, []string{f.Label})
		results = RunRunnableWithProgress(ctx, run, childReporter, []string{run.GetLabel()})
	}

	// If any child has an error, set the error on the parent
	if results.HasError() {
		result.Error = ErrResultChildrenHasError
		result.ExitCode = -1
		result.Status = ResultStatusError
	}

	return results
}
