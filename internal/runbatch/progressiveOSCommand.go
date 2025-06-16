// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"fmt"
	"time"

	"github.com/matt-FFFFFF/porch/internal/progress"
)

const (
	defaultProgressiveLogChannelBufferSize = 10 // Size of the log channel buffer
	defaultProgressiveLogUpdateInterval    = 500 * time.Millisecond
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
func (c *ProgressiveOSCommand) RunWithProgress(ctx context.Context, reporter progress.Reporter) Results {
	// Report that we're starting with our command label as the path
	reporter.Report(progress.Event{
		CommandPath: []string{c.GetLabel()}, // Use our label as the relative path
		Type:        progress.EventStarted,
		Message:     fmt.Sprintf("Starting %s", c.GetLabel()),
		Timestamp:   time.Now(),
	})

	logCh := make(chan string, defaultProgressiveLogChannelBufferSize) // Buffered channel for log messages
	defer close(logCh)                                                 // Ensure the channel is closed when done
	ctx = context.WithValue(ctx, ProgressiveLogChannelKey{}, (chan<- string)(logCh))
	ctx = context.WithValue(ctx, ProgressiveLogUpdateInterval{}, defaultProgressiveLogUpdateInterval) // Update every 500ms

	// This goroutine reads from the log channel and reports updates
	go func() {
		for {
			select {
			case <-ctx.Done():
				return // Exit if context is cancelled
			case logMsg, ok := <-logCh:
				if !ok {
					return // Channel closed, exit
				}
				// Report the log message as a progress event
				reporter.Report(progress.Event{
					CommandPath: []string{c.GetLabel()}, // Use our label as the relative path
					Type:        progress.EventProgress,
					Message:     fmt.Sprintf("Output from %s", c.GetLabel()),
					Timestamp:   time.Now(),
					Data: progress.EventData{
						OutputLine:      logMsg,
						ProgressMessage: fmt.Sprintf("Output from %s", c.GetLabel()),
					},
				})
			}
		}
	}()

	// Execute the original command
	results := c.Run(ctx)

	// Report completion based on results
	if len(results) > 0 {
		if results.HasError() {
			reporter.Report(progress.Event{
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
			reporter.Report(progress.Event{
				CommandPath: []string{c.GetLabel()}, // Use our label as the relative path
				Type:        progress.EventCompleted,
				Message:     fmt.Sprintf("Command completed: %s", c.GetLabel()),
				Timestamp:   time.Now(),
				Data: progress.EventData{
					ExitCode: results[0].ExitCode,
				},
			})
		}
	}

	return results
}

// Ensure ProgressiveOSCommand implements both interfaces.
var _ Runnable = (*ProgressiveOSCommand)(nil)
var _ ProgressiveRunnable = (*ProgressiveOSCommand)(nil)
