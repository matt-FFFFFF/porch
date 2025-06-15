// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
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
	RunWithProgress(ctx context.Context, reporter progress.ProgressReporter) Results
}

// ProgressContext wraps a context with progress reporting capabilities.
// It provides a convenient way to pass both context and progress reporting
// down through the command hierarchy.
type ProgressContext struct {
	context.Context
	Reporter    progress.ProgressReporter
	CommandPath []string
}

// NewProgressContext creates a new progress-aware context.
// The commandPath represents the hierarchical path to the current command
// in the execution tree (e.g., ["Build", "Quality Checks", "Unit Tests"]).
func NewProgressContext(ctx context.Context, reporter progress.ProgressReporter, commandPath []string) *ProgressContext {
	pathCopy := make([]string, len(commandPath))
	copy(pathCopy, commandPath)

	return &ProgressContext{
		Context:     ctx,
		Reporter:    reporter,
		CommandPath: pathCopy,
	}
}

// Child creates a child context with an extended command path.
// This is used when descending into nested commands to maintain
// the full hierarchical path for progress reporting.
func (pc *ProgressContext) Child(pathSegment string) *ProgressContext {
	newPath := make([]string, len(pc.CommandPath)+1)
	copy(newPath, pc.CommandPath)
	newPath[len(pc.CommandPath)] = pathSegment

	return &ProgressContext{
		Context:     pc.Context,
		Reporter:    pc.Reporter,
		CommandPath: newPath,
	}
}

// ReportStarted is a convenience method to report that a command has started.
func (pc *ProgressContext) ReportStarted(message string) {
	pc.Reporter.Report(progress.ProgressEvent{
		CommandPath: pc.CommandPath,
		Type:        progress.EventStarted,
		Message:     message,
		Timestamp:   time.Now(),
	})
}

// ReportProgress is a convenience method to report general progress.
func (pc *ProgressContext) ReportProgress(message string) {
	pc.Reporter.Report(progress.ProgressEvent{
		CommandPath: pc.CommandPath,
		Type:        progress.EventProgress,
		Message:     message,
		Timestamp:   time.Now(),
		Data: progress.EventData{
			ProgressMessage: message,
		},
	})
}

// ReportOutput is a convenience method to report command output.
func (pc *ProgressContext) ReportOutput(outputLine string, isStderr bool) {
	pc.Reporter.Report(progress.ProgressEvent{
		CommandPath: pc.CommandPath,
		Type:        progress.EventOutput,
		Message:     "Output received",
		Timestamp:   time.Now(),
		Data: progress.EventData{
			OutputLine: outputLine,
			IsStderr:   isStderr,
		},
	})
}

// ReportCompleted is a convenience method to report successful completion.
func (pc *ProgressContext) ReportCompleted(message string, exitCode int) {
	pc.Reporter.Report(progress.ProgressEvent{
		CommandPath: pc.CommandPath,
		Type:        progress.EventCompleted,
		Message:     message,
		Timestamp:   time.Now(),
		Data: progress.EventData{
			ExitCode: exitCode,
		},
	})
}

// ReportFailed is a convenience method to report command failure.
func (pc *ProgressContext) ReportFailed(message string, exitCode int, err error) {
	pc.Reporter.Report(progress.ProgressEvent{
		CommandPath: pc.CommandPath,
		Type:        progress.EventFailed,
		Message:     message,
		Timestamp:   time.Now(),
		Data: progress.EventData{
			ExitCode: exitCode,
			Error:    err,
		},
	})
}

// ReportSkipped is a convenience method to report that a command was skipped.
func (pc *ProgressContext) ReportSkipped(message string) {
	pc.Reporter.Report(progress.ProgressEvent{
		CommandPath: pc.CommandPath,
		Type:        progress.EventSkipped,
		Message:     message,
		Timestamp:   time.Now(),
	})
}

// RunRunnableWithProgress is a helper function that runs a Runnable with progress reporting.
// If the runnable implements ProgressiveRunnable, it uses RunWithProgress.
// Otherwise, it falls back to the regular Run method and synthesizes basic progress events.
func RunRunnableWithProgress(ctx context.Context, runnable Runnable, reporter progress.ProgressReporter, commandPath []string) Results {
	progressCtx := NewProgressContext(ctx, reporter, commandPath)

	// Check if the runnable supports progress reporting
	if progressive, ok := runnable.(ProgressiveRunnable); ok {
		return progressive.RunWithProgress(ctx, reporter)
	}

	// Fallback: run normally and synthesize basic events
	progressCtx.ReportStarted("Starting " + runnable.GetLabel())

	results := runnable.Run(ctx)

	// Report completion based on results
	if results.HasError() {
		progressCtx.ReportFailed("Command failed", -1, ErrResultChildrenHasError)
	} else {
		progressCtx.ReportCompleted("Command completed successfully", 0)
	}

	return results
}
