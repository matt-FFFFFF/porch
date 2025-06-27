// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"sync"
	"time"

	"github.com/matt-FFFFFF/porch/internal/progress"
)

// Ensure ParallelBatch implements ProgressiveRunnable.
var _ ProgressiveRunnable = (*ParallelBatch)(nil)

// RunWithProgress implements ProgressiveRunnable for ParallelBatch.
// It reuses the original execution logic but adds progress reporting for child commands.
func (b *ParallelBatch) RunWithProgress(ctx context.Context, reporter progress.Reporter) Results {
	// Report that this batch is starting
	ReportBatchStarted(reporter, b.Label, "parallel")

	// Create a child reporter for commands under this batch
	childReporter := CreateChildReporterForBatch(reporter, b.Label)

	// Create progressive versions of child commands for reporting
	progressiveCommands := wrapAsProgressive(b.Commands)

	// Execute with the original logic but with progressive commands and reporting
	results := b.executeWithProgressReporting(ctx, childReporter, progressiveCommands)

	// Report completion based on results
	ReportExecutionComplete(ctx, reporter, b.Label, results,
		"Parallel batch completed successfully",
		"Parallel batch failed")

	return results
}

// executeWithProgressReporting executes the batch using the original logic but with progress reporting.
func (b *ParallelBatch) executeWithProgressReporting(
	ctx context.Context, reporter progress.Reporter, progressiveCommands []Runnable,
) Results {
	children := make(Results, 0, len(progressiveCommands))
	wg := &sync.WaitGroup{}
	resChan := make(chan Results, len(progressiveCommands))
	hasError := false
	errorMutex := &sync.Mutex{}

	for _, cmd := range progressiveCommands {
		wg.Add(1)
		cmd.InheritEnv(b.Env)
		if err := cmd.SetCwd(b.Cwd); err != nil {
			// Report error setting cwd
			reporter.Report(progress.Event{
				CommandPath: []string{cmd.GetLabel()},
				Type:        progress.EventFailed,
				Message:     "Error setting working directory",
				Timestamp:   time.Now(),
				Data: progress.EventData{
					Error: err,
				},
			})

			children = append(children, &Result{
				Label:  cmd.GetLabel(),
				Status: ResultStatusError,
				Error:  err,
			})
			continue
		}

		go func(c Runnable) {
			defer wg.Done()

			// Execute the child command with progress reporting
			var childResults Results
			if progressive, ok := c.(ProgressiveRunnable); ok {
				childResults = progressive.RunWithProgress(ctx, reporter)
			} else {
				// Fallback to regular execution with simulated progress events
				reporter.Report(progress.Event{
					CommandPath: []string{c.GetLabel()},
					Type:        progress.EventStarted,
					Message:     "Starting command",
					Timestamp:   time.Now(),
				})

				childResults = c.Run(ctx)

				// Report completion using helper
				reportCommandExecution(reporter, c, childResults)
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
		children = append(children, r...)
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
