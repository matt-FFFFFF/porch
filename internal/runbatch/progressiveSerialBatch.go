// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"time"

	"github.com/matt-FFFFFF/porch/internal/progress"
)

// Ensure SerialBatch implements ProgressiveRunnable.
var _ ProgressiveRunnable = (*SerialBatch)(nil)

// RunWithProgress implements ProgressiveRunnable for SerialBatch.
// It reuses the original execution logic but adds progress reporting for child commands.
func (b *SerialBatch) RunWithProgress(ctx context.Context, reporter progress.Reporter) Results {
	// Our command path is just our label - the reporter will handle prefixing
	commandPath := []string{b.Label}

	// Report that this batch is starting
	reporter.Report(progress.Event{
		CommandPath: commandPath,
		Type:        progress.EventStarted,
		Message:     "Starting serial batch",
		Timestamp:   time.Now(),
	})

	// Create a child reporter for commands under this batch
	childReporter := NewChildReporter(reporter, []string{b.Label})

	// Execute with the original logic but with progressive commands and reporting
	results := b.executeWithProgressReporting(ctx, childReporter)

	// Report completion based on results
	if results.HasError() {
		reporter.Report(progress.Event{
			CommandPath: commandPath,
			Type:        progress.EventFailed,
			Message:     "Serial batch failed",
			Timestamp:   time.Now(),
			Data: progress.EventData{
				ExitCode: -1,
				Error:    ErrResultChildrenHasError,
			},
		})
	} else {
		reporter.Report(progress.Event{
			CommandPath: commandPath,
			Type:        progress.EventCompleted,
			Message:     "Serial batch completed successfully",
			Timestamp:   time.Now(),
			Data: progress.EventData{
				ExitCode: 0,
			},
		})
	}

	return results
}

// executeWithProgressReporting executes the batch using the original logic but with progress reporting.
func (b *SerialBatch) executeWithProgressReporting(ctx context.Context, reporter progress.Reporter) Results {
	results := make(Results, 0, len(b.Commands))
	newCwd := ""

	prevState := PreviousCommandStatus{
		State:    ResultStatusSuccess,
		ExitCode: 0,
		Err:      nil,
	}

	// Create progressive versions of child commands for reporting
	progressiveCommands := wrapAsProgressive(b.Commands)

OuterLoop:
	for i, cmd := range progressiveCommands { // Use progressive commands here!
		select {
		case <-ctx.Done():
			break OuterLoop
		default:
			// Inherit env and cwd from the batch if not already set
			cmd.InheritEnv(b.Env)
			cmd.SetCwd(b.Cwd, false)

			switch cmd.ShouldRun(prevState) {
			case ShouldRunActionSkip:
				// Report skipped command
				reporter.Report(progress.Event{
					CommandPath: []string{cmd.GetLabel()},
					Type:        progress.EventSkipped,
					Message:     "Command skipped intentionally",
					Timestamp:   time.Now(),
				})

				results = append(results, &Result{
					Label:  cmd.GetLabel(),
					Status: ResultStatusSkipped,
					Error:  ErrSkipIntentional,
				})
				continue OuterLoop

			case ShouldRunActionError:
				// Report skipped command due to error
				reporter.Report(progress.Event{
					CommandPath: []string{cmd.GetLabel()},
					Type:        progress.EventSkipped,
					Message:     "Command skipped due to previous error",
					Timestamp:   time.Now(),
				})

				results = append(results, &Result{
					Label:  cmd.GetLabel(),
					Status: ResultStatusSkipped,
					Error:  ErrSkipOnError,
				})
				continue OuterLoop
			}

			// Execute the child command with progress reporting
			var childResults Results
			if progressive, ok := cmd.(ProgressiveRunnable); ok {
				childResults = progressive.RunWithProgress(ctx, reporter)
			} else {
				// Fallback to regular execution with simulated progress events
				reporter.Report(progress.Event{
					CommandPath: []string{cmd.GetLabel()},
					Type:        progress.EventStarted,
					Message:     "Starting command",
					Timestamp:   time.Now(),
				})

				childResults = cmd.Run(ctx)

				// Report completion using helper
				reportCommandExecution(reporter, cmd, childResults)
			}

			prevState.State = childResults[0].Status
			prevState.ExitCode = childResults[0].ExitCode
			prevState.Err = childResults[0].Error

			newCwd = childResults[0].newCwd

			if newCwd != "" && i < len(progressiveCommands)-1 {
				// set the newCwd for the remaining commands in the batch
				for j := i + 1; j < len(progressiveCommands); j++ {
					progressiveCommands[j].SetCwd(newCwd, true)
				}
			}

			results = append(results, childResults...)
		}
	}

	res := Results{&Result{
		Label:    b.Label,
		ExitCode: 0,
		Error:    nil,
		Children: results,
		Status:   ResultStatusSuccess,
	}}
	if results.HasError() {
		res[0].ExitCode = -1
		res[0].Error = ErrResultChildrenHasError
		res[0].Status = ResultStatusError
	}

	return res
}
