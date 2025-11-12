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

// RunWithProgress implements ProgressiveRunnable.RunWithProgress.
// It executes the OS command while providing real-time progress updates.
func (c *OSCommand) RunWithProgress(ctx context.Context, reporter progress.Reporter) Results {
	// Report that we're starting
	ReportCommandStarted(reporter, c.GetLabel())

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

	// Report completion based on results using helper
	ReportExecutionComplete(ctx, reporter, c.GetLabel(), results,
		fmt.Sprintf("Command completed: %s", c.GetLabel()),
		fmt.Sprintf("Command failed: %s", c.GetLabel()))

	return results
}

// Ensure ProgressiveOSCommand implements both interfaces.
var _ Runnable = (*OSCommand)(nil)
var _ ProgressiveRunnable = (*OSCommand)(nil)
