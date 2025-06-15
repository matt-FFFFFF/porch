// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package progress

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventType_String(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		expected  string
	}{
		{
			name:      "EventStarted",
			eventType: EventStarted,
			expected:  "started",
		},
		{
			name:      "EventProgress",
			eventType: EventProgress,
			expected:  "progress",
		},
		{
			name:      "EventOutput",
			eventType: EventOutput,
			expected:  "output",
		},
		{
			name:      "EventCompleted",
			eventType: EventCompleted,
			expected:  "completed",
		},
		{
			name:      "EventFailed",
			eventType: EventFailed,
			expected:  "failed",
		},
		{
			name:      "EventSkipped",
			eventType: EventSkipped,
			expected:  "skipped",
		},
		{
			name:      "Unknown event type",
			eventType: EventType(999),
			expected:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.eventType.String())
		})
	}
}

func TestNullReporter(t *testing.T) {
	reporter := NewNullReporter()
	require.NotNil(t, reporter)

	// These should not panic
	reporter.Report(Event{
		CommandPath: []string{"test"},
		Type:        EventStarted,
		Message:     "test message",
		Timestamp:   time.Now(),
	})

	reporter.Close()
}

func TestChannelReporter(t *testing.T) {
	ctx := context.Background()
	reporter := NewChannelReporter(ctx, 10)
	require.NotNil(t, reporter)

	// Test reporting events
	event := Event{
		CommandPath: []string{"test", "command"},
		Type:        EventStarted,
		Message:     "Test started",
		Timestamp:   time.Now(),
	}

	reporter.Report(event)

	// Test receiving events
	select {
	case receivedEvent := <-reporter.Events():
		assert.Equal(t, event.CommandPath, receivedEvent.CommandPath)
		assert.Equal(t, event.Type, receivedEvent.Type)
		assert.Equal(t, event.Message, receivedEvent.Message)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Event not received within timeout")
	}

	reporter.Close()

	// Test that closed reporter drops events
	reporter.Report(Event{
		Type:    EventCompleted,
		Message: "Should be dropped",
	})
}

func TestChannelReporter_BufferOverflow(t *testing.T) {
	ctx := context.Background()
	// Create reporter with small buffer
	reporter := NewChannelReporter(ctx, 1)
	require.NotNil(t, reporter)

	// Fill the buffer
	reporter.Report(Event{Type: EventStarted, Message: "Event 1"})

	// This should not block due to the non-blocking send
	reporter.Report(Event{Type: EventProgress, Message: "Event 2"})

	reporter.Close()
}

type mockListener struct {
	events []Event
}

func (ml *mockListener) OnEvent(event Event) {
	ml.events = append(ml.events, event)
}

func TestChannelReporter_Listen(t *testing.T) {
	ctx := context.Background()
	reporter := NewChannelReporter(ctx, 10)
	require.NotNil(t, reporter)

	listener := &mockListener{}
	reporter.Listen(listener)

	// Send some events
	events := []Event{
		{Type: EventStarted, Message: "Started"},
		{Type: EventProgress, Message: "Progress"},
		{Type: EventCompleted, Message: "Completed"},
	}

	for _, event := range events {
		reporter.Report(event)
	}

	// Give the listener goroutine time to process
	time.Sleep(10 * time.Millisecond)

	reporter.Close()

	// Check that all events were received
	assert.Len(t, listener.events, len(events))

	for i, expectedEvent := range events {
		assert.Equal(t, expectedEvent.Type, listener.events[i].Type)
		assert.Equal(t, expectedEvent.Message, listener.events[i].Message)
	}
}
