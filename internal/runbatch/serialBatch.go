// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"slices"
	"time"

	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/progress"
)

var _ Runnable = (*SerialBatch)(nil)

const (
	// SerialBatchType is the type identifier for SerialBatch runnables.
	SerialBatchType = "SerialBatch"
)

// SerialBatch represents a collection of commands, which are run serially.
type SerialBatch struct {
	*BaseCommand
	Commands []Runnable // The commands or nested batches to run
}

// Run implements the Runnable interface for SerialBatch.
func (b *SerialBatch) Run(ctx context.Context) Results {
	label := FullLabel(b)
	logger := ctxlog.Logger(ctx).
		With("label", label).
		With("runnableType", "SerialBatch")

	// Report that this batch is starting if we have a reporter
	if b.hasProgressReporter() {
		ReportBatchStarted(b.GetProgressReporter(), b.Label, "serial")
	}

	// Propagate reporter to child commands
	PropagateReporterToChildren(b.GetProgressReporter(), b.Label, b.Commands)

	results := make(Results, 0, len(b.Commands))
	newCwd := ""

	prevState := CommandStatus{
		State:    ResultStatusSuccess,
		ExitCode: 0,
		Err:      nil,
	}

OuterLoop:
	for i, cmd := range slices.All(b.Commands) {
		select {
		case <-ctx.Done():
			break OuterLoop
		default:
			// Inherit env and cwd from the batch if not already set
			logger.Debug("setting environment for child commands",
				"commandLabel", cmd.GetLabel(),
				"env", b.Env)
			cmd.InheritEnv(b.Env)

			switch cmd.ShouldRun(prevState) {
			case ShouldRunActionSkip:
				// Report skipped command if we have a reporter
				if b.hasProgressReporter() {
					b.GetProgressReporter().Report(progress.Event{
						CommandPath: []string{cmd.GetLabel()},
						Type:        progress.EventSkipped,
						Message:     "Command skipped intentionally",
						Timestamp:   time.Now(),
						Data: progress.EventData{
							Error: ErrSkipIntentional,
						},
					})
				}

				results = append(results, &Result{
					Label:  cmd.GetLabel(),
					Status: ResultStatusSkipped,
					Error:  ErrSkipIntentional,
					Cwd:    cmd.GetCwd(),
					Type:   cmd.GetType(),
				})

				continue OuterLoop

			case ShouldRunActionError:
				// Report skipped command due to error if we have a reporter
				if b.hasProgressReporter() {
					b.GetProgressReporter().Report(progress.Event{
						CommandPath: []string{cmd.GetLabel()},
						Type:        progress.EventSkipped,
						Message:     "Command skipped due to previous error",
						Timestamp:   time.Now(),
						Data: progress.EventData{
							Error: ErrSkipOnError,
						},
					})
				}

				results = append(results, &Result{
					Label:  cmd.GetLabel(),
					Status: ResultStatusSkipped,
					Error:  ErrSkipOnError,
					Cwd:    cmd.GetCwd(),
					Type:   cmd.GetType(),
				})

				continue OuterLoop
			}

			childResults := cmd.Run(ctx)

			prevState.State = childResults[0].Status
			prevState.ExitCode = childResults[0].ExitCode
			prevState.Err = childResults[0].Error

			newCwd = childResults[0].newCwd

			if newCwd != "" && i < len(b.Commands)-1 {
				logger.Debug("newCwd is set, updating working directory on parent command",
					"newCwd", newCwd,
				)

				// set the newCwd for the remaining commands in the batch
				// by updating the parent serial batch cwd.
				b.cwd = newCwd
			}

			results = slices.Concat(results, childResults)
		}
	}

	res := Results{&Result{
		Label:    b.Label,
		ExitCode: 0,
		Error:    nil,
		StdOut:   nil,
		StdErr:   nil,
		Children: results,
		Status:   ResultStatusSuccess,
		Cwd:      b.GetCwd(),
		Type:     b.GetType(),
	}}
	if results.HasError() {
		res[0].ExitCode = -1
		res[0].Error = ErrResultChildrenHasError
		res[0].Status = ResultStatusError
	}

	// Report completion based on results if we have a reporter
	if b.hasProgressReporter() {
		ReportExecutionComplete(ctx, b.GetProgressReporter(), b.Label, res,
			"Serial batch completed successfully",
			"Serial batch failed")
	}

	return res
}

// SetProgressReporter sets the progress reporter and propagates it to all child commands.
func (b *SerialBatch) SetProgressReporter(reporter progress.Reporter) {
	b.BaseCommand.SetProgressReporter(reporter)
	// Note: We don't propagate here as it's done in Run() with a child reporter
}

// GetType returns the type of the runnable.
func (b *SerialBatch) GetType() string {
	return SerialBatchType
}
