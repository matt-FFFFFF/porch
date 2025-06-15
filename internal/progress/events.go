// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package progress

import (
	"time"
)

// Event represents a real-time update from command execution.
// Events are emitted throughout the command lifecycle to provide
// real-time feedback for TUI and other monitoring systems.
type Event struct {
	CommandPath []string  // Hierarchical path to command (e.g., ["Build", "Quality Checks", "Unit Tests"])
	Type        EventType // Event type indicating what happened
	Message     string    // Human-readable status message
	Timestamp   time.Time // When the event occurred
	Data        EventData // Type-specific data
}

// EventType represents the type of progress event.
type EventType int

const (
	// EventStarted indicates a command has begun execution.
	EventStarted EventType = iota
	// EventProgress indicates general progress information.
	EventProgress
	// EventCompleted indicates successful completion.
	EventCompleted
	// EventFailed indicates the command failed.
	EventFailed
	// EventSkipped indicates the command was skipped due to conditions.
	EventSkipped
)

// String implements the Stringer interface for EventType.
func (et EventType) String() string {
	switch et {
	case EventStarted:
		return "started"
	case EventProgress:
		return "progress"
	case EventCompleted:
		return "completed"
	case EventFailed:
		return "failed"
	case EventSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// EventData contains type-specific information for progress events.
type EventData struct {
	// For EventOutput
	OutputLine string // The actual output line
	IsStderr   bool   // True if this is stderr output

	// For EventCompleted/EventFailed
	ExitCode int   // Command exit code
	Error    error // Error if the command failed

	// For EventProgress
	ProgressMessage string // Additional progress information
}

// Reporter is the interface for sending progress events.
// Commands implement this to emit real-time updates during execution.
type Reporter interface {
	// Report sends a progress event. Implementations should be non-blocking
	// and handle the case where the receiver might not be listening.
	Report(event Event)
	// Close signals that no more events will be sent and cleans up resources.
	Close()
}

// Listener receives progress events from commands.
// TUI implementations and other monitoring systems implement this interface.
type Listener interface {
	// OnEvent is called when a progress event is received.
	// Implementations should handle events quickly to avoid blocking
	// the reporting goroutine.
	OnEvent(event Event)
}

// NullReporter is a no-op implementation of Reporter.
// Used when progress reporting is not needed.
type NullReporter struct{}

// Report implements Reporter.Report by doing nothing.
func (nr *NullReporter) Report(_ Event) {
	// No-op
}

// Close implements Reporter.Close by doing nothing.
func (nr *NullReporter) Close() {
	// No-op
}

// NewNullReporter creates a new NullReporter.
func NewNullReporter() Reporter {
	return &NullReporter{}
}
