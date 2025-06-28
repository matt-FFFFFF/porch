// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"time"

	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/progress"
)

// Ensure ForEachCommand implements ProgressiveRunnable.
var _ ProgressiveRunnable = (*ForEachCommand)(nil)

// RunWithProgress implements ProgressiveRunnable for ForEachCommand.
func (f *ForEachCommand) RunWithProgress(ctx context.Context, reporter progress.Reporter) Results {
	results := f.runWithProgressiveChildren(ctx, reporter)
	// Don't report our own start/completion events - we want to be transparent
	// in the progress hierarchy and let the internal batch report directly.

	// Use the original Run method but with progressive reporting
	// We'll override the child commands to be progressive if possible

	// Don't report completion - let the internal batch handle all reporting
	return results
}

// runWithProgressiveChildren runs the foreach command using the original logic
// but with progressive reporting for child batches.
func (f *ForEachCommand) runWithProgressiveChildren(ctx context.Context, reporter progress.Reporter) Results {
	label := FullLabel(f)
	logger := ctxlog.Logger(ctx).
		With("label", label).
		With("runnableType", "progressiveForEachCommand")

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
		for _, skipErr := range f.ItemsSkipOnErrors {
			// If the error is in the skip list, treat it as a skipped result.
			if errors.Is(err, skipErr) {
				result.Status = ResultStatusSkipped
				result.Error = ErrSkipIntentional
				reporter.Report(progress.Event{
					CommandPath: []string{f.Label},
					Type:        progress.EventSkipped,
					Message:     result.Error.Error(),
					Timestamp:   time.Now(),
					Data: progress.EventData{
						ExitCode:   result.ExitCode,
						Error:      result.Error,
						OutputLine: fmt.Sprintf("%v: %v", ErrSkipIntentional, err),
					},
				})

				return Results{result}
			}
		}

		logger.Debug("items to iterate over",
			"count", len(items),
			"items", items)

		// If the error is not in the skip list, return an error result.
		result.Error = fmt.Errorf("%w: %v", ErrItemsProviderFailed, err)
		result.Status = ResultStatusError
		result.ExitCode = -1

		reporter.Report(progress.Event{
			CommandPath: []string{f.Label},
			Type:        progress.EventFailed,
			Message:     result.Error.Error(),
			Timestamp:   time.Now(),
			Data: progress.EventData{
				ExitCode: result.ExitCode,
				Error:    result.Error,
			},
		})

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
		}

		newEnv[ItemEnvVar] = item
		base := NewBaseCommand(
			fmt.Sprintf("[%s]", item),
			cwd,
			f.CwdRel,
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

		switch f.CwdStrategy {
		case CwdStrategyItemRelative:
			serialBatch.SetCwd(item)
		}

		foreachCommands[i] = serialBatch
	}

	base := NewBaseCommand(
		f.Label, f.Cwd, f.CwdRel, f.RunsOnCondition, f.RunsOnExitCodes, maps.Clone(f.Env),
	)
	base.SetParent(f.GetParent())

	// Handle different execution modes with progressive execution
	var run Runnable

	switch f.Mode {
	case ForEachParallel:
		base.Label = f.Label + " (parallel)"
		run = &ParallelBatch{
			BaseCommand: base,
			Commands:    foreachCommands,
		}
	case ForEachSerial:
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
		// Use a transparent reporter so the batch reports directly without the ForEach layer
		transparentReporter := NewTransparentReporter(reporter)
		results = progressive.RunWithProgress(ctx, transparentReporter)
	} else {
		// Fallback to regular execution with transparent progress events
		transparentReporter := NewTransparentReporter(reporter)
		results = RunRunnableWithProgress(ctx, run, transparentReporter, []string{run.GetLabel()})
	}

	// If any child has an error, set the error on the parent
	if results.HasError() {
		result.Error = ErrResultChildrenHasError
		result.ExitCode = -1
		result.Status = ResultStatusError
	}

	return results
}
