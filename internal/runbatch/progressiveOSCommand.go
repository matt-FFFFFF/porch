// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/matt-FFFFFF/porch/internal/progress"
)

// ProgressiveOSCommand extends OSCommand with progress reporting capabilities.
// It implements both Runnable and ProgressiveRunnable interfaces.
type ProgressiveOSCommand struct {
	*OSCommand
}

// NewProgressiveOSCommand creates a new progressive OS command.
func NewProgressiveOSCommand(osCommand *OSCommand) *ProgressiveOSCommand {
	return &ProgressiveOSCommand{
		OSCommand: osCommand,
	}
}

// RunWithProgress implements ProgressiveRunnable.RunWithProgress.
// It executes the OS command while providing real-time progress updates.
func (c *ProgressiveOSCommand) RunWithProgress(ctx context.Context, reporter progress.ProgressReporter) Results {
	// Report that we're starting with our command label as the path
	reporter.Report(progress.ProgressEvent{
		CommandPath: []string{c.GetLabel()}, // Use our label as the relative path
		Type:        progress.EventStarted,
		Message:     fmt.Sprintf("Starting %s", c.GetLabel()),
		Timestamp:   time.Now(),
	})

	// Execute the original command
	results := c.OSCommand.Run(ctx)

	// Report completion based on results
	if len(results) > 0 {
		if results.HasError() {
			reporter.Report(progress.ProgressEvent{
				CommandPath: []string{c.GetLabel()}, // Use our label as the relative path
				Type:        progress.EventFailed,
				Message:     fmt.Sprintf("Command failed: %s", c.GetLabel()),
				Timestamp:   time.Now(),
				Data: progress.EventData{
					ExitCode: results[0].ExitCode,
					Error:    results[0].Error,
				},
			})
		} else {
			reporter.Report(progress.ProgressEvent{
				CommandPath: []string{c.GetLabel()}, // Use our label as the relative path
				Type:        progress.EventCompleted,
				Message:     fmt.Sprintf("Command completed: %s", c.GetLabel()),
				Timestamp:   time.Now(),
				Data: progress.EventData{
					ExitCode: results[0].ExitCode,
				},
			})
		}

		// Report output if available
		if len(results[0].StdOut) > 0 {
			outputLines := strings.Split(string(results[0].StdOut), "\n")
			for _, line := range outputLines {
				if strings.TrimSpace(line) != "" {
					reporter.Report(progress.ProgressEvent{
						CommandPath: []string{c.GetLabel()}, // Use our label as the relative path
						Type:        progress.EventOutput,
						Timestamp:   time.Now(),
						Data: progress.EventData{
							OutputLine: strings.TrimSpace(line),
							IsStderr:   false,
						},
					})
				}
			}
		}
	}

	return results
}

// Ensure ProgressiveOSCommand implements both interfaces
var _ Runnable = (*ProgressiveOSCommand)(nil)
var _ ProgressiveRunnable = (*ProgressiveOSCommand)(nil)
