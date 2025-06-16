// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"fmt"
	"time"

	"github.com/matt-FFFFFF/porch/internal/progress"
)

// ProgressiveLogChannelKey is used to store the channel in the Runnable context
// for real-time progress updates. This allows commands to report their
// logging output to the progressive TUI in real-time.
type ProgressiveLogChannelKey struct{}

// ProgressiveLogUpdateInterval defines the interval in seconds at which
// the progressive TUI updates its display with new log output.
type ProgressiveLogUpdateInterval struct{}

// ProgressiveRunnable extends Runnable with progress reporting capabilities.
// Commands that implement this interface can provide real-time updates
// during execution while maintaining compatibility with the existing
// Runnable interface.
type ProgressiveRunnable interface {
	Runnable
	// RunWithProgress executes the command while reporting progress events.
	// The reporter receives real-time updates about command execution state.
	RunWithProgress(ctx context.Context, reporter progress.Reporter) Results
}

// reportCommandExecution is a helper function that reports command execution results
// consistently across different progressive implementations.
func reportCommandExecution(reporter progress.Reporter, cmd Runnable, results Results) {
	commandPath := []string{cmd.GetLabel()}

	if results.HasError() {
		exitCode := -1
		err := ErrResultChildrenHasError

		if len(results) > 0 {
			exitCode = results[0].ExitCode
			err = results[0].Error
		}

		reporter.Report(progress.Event{
			CommandPath: commandPath,
			Type:        progress.EventFailed,
			Message:     fmt.Sprintf("Command failed: %s", cmd.GetLabel()),
			Timestamp:   time.Now(),
			Data: progress.EventData{
				ExitCode: exitCode,
				Error:    err,
			},
		})

		return
	}

	var exitCode int
	if len(results) > 0 {
		exitCode = results[0].ExitCode
	}

	reporter.Report(progress.Event{
		CommandPath: commandPath,
		Type:        progress.EventCompleted,
		Message:     fmt.Sprintf("Command completed successfully: %s", cmd.GetLabel()),
		Timestamp:   time.Now(),
		Data: progress.EventData{
			ExitCode: exitCode,
		},
	})
}

// wrapAsProgressive converts regular commands to progressive commands where possible.
// This helper function consolidates the logic for wrapping non-progressive commands
// with progress reporting capabilities.
func wrapAsProgressive(commands []Runnable) []Runnable {
	progressiveCommands := make([]Runnable, len(commands))

	for i, cmd := range commands {
		if progressive, ok := cmd.(ProgressiveRunnable); ok {
			// Already progressive, use as-is
			progressiveCommands[i] = progressive
			continue
		}
		// Wrap non-progressive commands
		if osCmd, ok := cmd.(*OSCommand); ok {
			progressiveCommands[i] = NewProgressiveOSCommand(osCmd)
			continue
		}
		// For other command types, use original
		progressiveCommands[i] = cmd
	}

	return progressiveCommands
}

// RunRunnableWithProgress is a helper function that runs a Runnable with progress reporting.
// If the runnable implements ProgressiveRunnable, it uses RunWithProgress.
// Otherwise, it falls back to the regular Run method and synthesizes basic progress events.
func RunRunnableWithProgress(
	ctx context.Context, runnable Runnable, reporter progress.Reporter, commandPath []string,
) Results {
	// Check if the runnable supports progress reporting
	if progressive, ok := runnable.(ProgressiveRunnable); ok {
		// Create a child reporter with the provided command path for proper hierarchical context
		childReporter := NewChildReporter(reporter, commandPath)
		return progressive.RunWithProgress(ctx, childReporter)
	}

	// Fallback: run normally and synthesize basic events
	reporter.Report(progress.Event{
		CommandPath: commandPath,
		Type:        progress.EventStarted,
		Message:     "Starting " + runnable.GetLabel(),
		Timestamp:   time.Now(),
	})

	results := runnable.Run(ctx)

	// Report completion based on results
	if results.HasError() {
		reporter.Report(progress.Event{
			CommandPath: commandPath,
			Type:        progress.EventFailed,
			Message:     "Command failed",
			Timestamp:   time.Now(),
			Data: progress.EventData{
				ExitCode: -1,
				Error:    ErrResultChildrenHasError,
			},
		})

		return results
	}

	reporter.Report(progress.Event{
		CommandPath: commandPath,
		Type:        progress.EventCompleted,
		Message:     "Command completed successfully",
		Timestamp:   time.Now(),
		Data: progress.EventData{
			ExitCode: 0,
		},
	})

	return results
}

// ProgressReportingHelpers - Shared progress reporting functions
// These helpers consolidate common patterns across progressive implementations.

// ReportBatchStarted reports that a batch operation has started.
func ReportBatchStarted(reporter progress.Reporter, label, batchType string) {
	reporter.Report(progress.Event{
		CommandPath: []string{label},
		Type:        progress.EventStarted,
		Message:     fmt.Sprintf("Starting %s batch", batchType),
		Timestamp:   time.Now(),
	})
}

// ReportCommandStarted reports that a command has started.
func ReportCommandStarted(reporter progress.Reporter, label string) {
	reporter.Report(progress.Event{
		CommandPath: []string{label},
		Type:        progress.EventStarted,
		Message:     fmt.Sprintf("Starting %s", label),
		Timestamp:   time.Now(),
	})
}

// ReportExecutionComplete reports command/batch completion based on results.
// It handles both success and failure cases with appropriate event data.
func ReportExecutionComplete(reporter progress.Reporter, label string, results Results, successMsg, failureMsg string) {
	commandPath := []string{label}

	if results.HasError() {
		exitCode := -1
		err := ErrResultChildrenHasError

		if len(results) > 0 {
			exitCode = results[0].ExitCode
			err = results[0].Error
		}

		reporter.Report(progress.Event{
			CommandPath: commandPath,
			Type:        progress.EventFailed,
			Message:     failureMsg,
			Timestamp:   time.Now(),
			Data: progress.EventData{
				ExitCode: exitCode,
				Error:    err,
			},
		})

		return
	}

	var exitCode int
	if len(results) > 0 {
		exitCode = results[0].ExitCode
	}

	reporter.Report(progress.Event{
		CommandPath: commandPath,
		Type:        progress.EventCompleted,
		Message:     successMsg,
		Timestamp:   time.Now(),
		Data: progress.EventData{
			ExitCode: exitCode,
		},
	})
}

// CreateChildReporterForBatch creates a child reporter for batch operations.
// This consolidates the common pattern of creating child reporters with batch labels.
func CreateChildReporterForBatch(parent progress.Reporter, batchLabel string) progress.Reporter {
	return NewChildReporter(parent, []string{batchLabel})
}
