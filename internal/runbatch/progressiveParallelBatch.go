// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/matt-FFFFFF/porch/internal/progress"
)

// Ensure ParallelBatch implements ProgressiveRunnable
var _ ProgressiveRunnable = (*ParallelBatch)(nil)

// RunWithProgress implements ProgressiveRunnable for ParallelBatch.
// It reuses the original execution logic but adds progress reporting for child commands.
func (b *ParallelBatch) RunWithProgress(ctx context.Context, reporter progress.ProgressReporter) Results {
	// Our command path is just our label - the reporter will handle prefixing
	commandPath := []string{b.Label}

	// Report that this batch is starting
	reporter.Report(progress.ProgressEvent{
		CommandPath: commandPath,
		Type:        progress.EventStarted,
		Message:     "Starting parallel batch",
		Timestamp:   time.Now(),
	})

	// Create a child reporter for commands under this batch
	childReporter := NewChildReporter(reporter, []string{b.Label})

	// Create progressive versions of child commands for reporting
	progressiveCommands := make([]Runnable, len(b.Commands))
	for i, cmd := range b.Commands {
		if progressive, ok := cmd.(ProgressiveRunnable); ok {
			// Already progressive, use as-is
			progressiveCommands[i] = progressive
		} else {
			// Wrap non-progressive commands
			if osCmd, ok := cmd.(*OSCommand); ok {
				progressiveCommands[i] = NewProgressiveOSCommand(osCmd)
			} else {
				// For other command types, use original
				progressiveCommands[i] = cmd
			}
		}
	}

	// Execute with the original logic but with progressive commands and reporting
	results := b.executeWithProgressReporting(ctx, childReporter, progressiveCommands)

	// Report completion based on results
	if results.HasError() {
		reporter.Report(progress.ProgressEvent{
			CommandPath: commandPath,
			Type:        progress.EventFailed,
			Message:     "Parallel batch failed",
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
			Message:     "Parallel batch completed successfully",
			Timestamp:   time.Now(),
			Data: progress.EventData{
				ExitCode: 0,
			},
		})
	}

	return results
}

// executeWithProgressReporting executes the batch using the original logic but with progress reporting
func (b *ParallelBatch) executeWithProgressReporting(ctx context.Context, reporter progress.ProgressReporter, progressiveCommands []Runnable) Results {
	children := make(Results, 0, len(progressiveCommands))
	wg := &sync.WaitGroup{}
	resChan := make(chan Results, len(progressiveCommands))
	hasError := false
	errorMutex := &sync.Mutex{}

	for _, cmd := range progressiveCommands {
		wg.Add(1)
		cmd.InheritEnv(b.Env)
		cmd.SetCwd(b.Cwd, false)

		go func(c Runnable) {
			defer wg.Done()

			// Execute the child command with progress reporting
			var childResults Results
			if progressive, ok := c.(ProgressiveRunnable); ok {
				childResults = progressive.RunWithProgress(ctx, reporter)
			} else {
				// Fallback to regular execution with simulated progress events
				reporter.Report(progress.ProgressEvent{
					CommandPath: []string{c.GetLabel()},
					Type:        progress.EventStarted,
					Message:     "Starting command",
					Timestamp:   time.Now(),
				})

				childResults = c.Run(ctx)

				// Report completion
				if childResults.HasError() {
					reporter.Report(progress.ProgressEvent{
						CommandPath: []string{c.GetLabel()},
						Type:        progress.EventFailed,
						Message:     "Command failed",
						Timestamp:   time.Now(),
						Data: progress.EventData{
							ExitCode: childResults[0].ExitCode,
							Error:    childResults[0].Error,
						},
					})
				} else {
					reporter.Report(progress.ProgressEvent{
						CommandPath: []string{c.GetLabel()},
						Type:        progress.EventCompleted,
						Message:     "Command completed successfully",
						Timestamp:   time.Now(),
						Data: progress.EventData{
							ExitCode: childResults[0].ExitCode,
						},
					})
				}
			}

			// Check for errors in a thread-safe way
			if childResults.HasError() {
				errorMutex.Lock()
				hasError = true
				errorMutex.Unlock()
			}

			resChan <- childResults
		}(cmd)
	}

	wg.Wait()
	close(resChan)

	for r := range resChan {
		children = slices.Concat(children, r)
	}

	res := Results{&Result{
		Label:    b.Label,
		Children: children,
		Status:   ResultStatusSuccess,
	}}

	if hasError {
		res[0].ExitCode = -1
		res[0].Error = ErrResultChildrenHasError
		res[0].Status = ResultStatusError
	}

	return res
}
