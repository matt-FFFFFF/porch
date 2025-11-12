// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package runbatch provides helper functions for progress reporting.
package runbatch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/progress"
)

// ReportBatchStarted reports that a batch operation has started.
// If reporter is nil, this is a no-op.
func ReportBatchStarted(reporter progress.Reporter, label, batchType string) {
	if reporter == nil {
		return
	}

	reporter.Report(progress.Event{
		CommandPath: []string{label},
		Type:        progress.EventStarted,
		Message:     fmt.Sprintf("Starting %s batch", batchType),
		Timestamp:   time.Now(),
	})
}

// ReportCommandStarted reports that a command has started.
// If reporter is nil, this is a no-op.
func ReportCommandStarted(reporter progress.Reporter, label string) {
	if reporter == nil {
		return
	}

	reporter.Report(progress.Event{
		CommandPath: []string{label},
		Type:        progress.EventStarted,
		Message:     fmt.Sprintf("Starting %s", label),
		Timestamp:   time.Now(),
	})
}

// ReportExecutionComplete reports command/batch completion based on results.
// It handles both success and failure cases with appropriate event data.
// If reporter is nil, this is a no-op.
func ReportExecutionComplete(
	ctx context.Context,
	reporter progress.Reporter,
	label string, results Results,
	successMsg, failureMsg string) {
	if reporter == nil {
		return
	}

	logger := ctxlog.Logger(ctx)
	commandPath := []string{label}

	if results.HasError() {
		exitCode := -1
		err := ErrResultChildrenHasError

		outputline := ""

		if len(results) > 0 {
			exitCode = results[0].ExitCode
			err = results[0].Error
			firstErrLine, _, _ := strings.Cut(string(results[0].StdErr), "\n")

			if firstErrLine != "" {
				outputline = firstErrLine
			}
		}

		logger.Debug("ReportExecutionComplete: Reporting failed command",
			"label", label,
			"commandPath", commandPath,
			"exitCode", exitCode,
			"resultsLength", len(results))

		reporter.Report(progress.Event{
			CommandPath: commandPath,
			Type:        progress.EventFailed,
			Message:     failureMsg,
			Timestamp:   time.Now(),
			Data: progress.EventData{
				ExitCode:   exitCode,
				Error:      err,
				IsStderr:   true,
				OutputLine: outputline,
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
// If parent is nil, returns nil.
func CreateChildReporterForBatch(parent progress.Reporter, batchLabel string) progress.Reporter {
	if parent == nil {
		return nil
	}

	return NewChildReporter(parent, []string{batchLabel})
}

// PropagateReporterToChildren propagates a parent reporter to all child commands in a batch.
// This is extracted to avoid duplication between SerialBatch and ParallelBatch.
// If parent is nil, this is a no-op.
func PropagateReporterToChildren(parent progress.Reporter, batchLabel string, commands []Runnable) {
	if parent == nil {
		return
	}

	childReporter := CreateChildReporterForBatch(parent, batchLabel)
	for _, cmd := range commands {
		cmd.SetProgressReporter(childReporter)
	}
}
