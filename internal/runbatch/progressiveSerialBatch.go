// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"slices"
	"time"

	"github.com/matt-FFFFFF/porch/internal/progress"
)

// Ensure SerialBatch implements ProgressiveRunnable.
var _ ProgressiveRunnable = (*SerialBatch)(nil)

// RunWithProgress implements ProgressiveRunnable for SerialBatch.
// It reuses the original execution logic but adds progress reporting for child commands.
func (b *SerialBatch) RunWithProgress(ctx context.Context, reporter progress.Reporter) Results {
	// Report that this batch is starting
	ReportBatchStarted(reporter, b.Label, "serial")

	// Create a child reporter for commands under this batch
	childReporter := CreateChildReporterForBatch(reporter, b.Label)

	// Execute with the original logic but with progressive commands and reporting
	results := b.executeWithProgressReporting(ctx, childReporter)

	// Report completion based on results
	ReportExecutionComplete(ctx, reporter, b.Label, results,
		"Serial batch completed successfully",
		"Serial batch failed")

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
					Data: progress.EventData{
						Error: ErrSkipIntentional,
					},
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
					Data: progress.EventData{
						Error: ErrSkipOnError,
					},
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
				for rb := range slices.Values(progressiveCommands[i+1:]) {
					rb.SetCwd(newCwd, true)
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
