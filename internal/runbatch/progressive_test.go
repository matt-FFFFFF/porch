// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/progress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRunnableWithProgress_ProgressiveRunnable(t *testing.T) {
	ctx := context.Background()
	reporter := progress.NewChannelReporter(ctx, 10)

	defer reporter.Close()

	mock := &MockProgressiveRunnable{
		BaseCommand: &BaseCommand{
			Label: "test-command",
		},
	}

	commandPath := []string{"parent", "child"}
	results := RunRunnableWithProgress(ctx, mock, reporter, commandPath)

	require.Len(t, results, 1)
	assert.Equal(t, "test-command", results[0].Label)
	assert.Equal(t, ResultStatusSuccess, results[0].Status)
}

func TestRunRunnableWithProgress_FallbackRunnable(t *testing.T) {
	ctx := context.Background()
	reporter := progress.NewChannelReporter(ctx, 10)

	defer reporter.Close()

	// Create a simple mock runnable that doesn't implement ProgressiveRunnable
	mock := &SimpleMockRunnable{
		BaseCommand: &BaseCommand{
			Label: "regular-command",
		},
	}

	commandPath := []string{"parent", "child"}
	results := RunRunnableWithProgress(ctx, mock, reporter, commandPath)

	require.Len(t, results, 1)
	assert.Equal(t, "regular-command", results[0].Label)

	// Verify fallback events were sent
	events := make([]progress.Event, 0, 2)

OuterLoop:
	for len(events) < 2 {
		select {
		case event := <-reporter.Events():
			events = append(events, event)
		default:
			break OuterLoop
		}
	}

	require.Len(t, events, 2)
	assert.Equal(t, progress.EventStarted, events[0].Type)
	assert.Equal(t, commandPath, events[0].CommandPath)
	assert.Equal(t, "Starting regular-command", events[0].Message)

	assert.Equal(t, progress.EventCompleted, events[1].Type)
	assert.Equal(t, commandPath, events[1].CommandPath)
	assert.Equal(t, "Command completed successfully", events[1].Message)
}

// SimpleMockRunnable is a basic implementation of Runnable for testing.
type SimpleMockRunnable struct {
	*BaseCommand
}

func (s *SimpleMockRunnable) Run(ctx context.Context) Results {
	return Results{&Result{
		Label:  s.Label,
		Status: ResultStatusSuccess,
	}}
}

func TestProgressiveRunnableInterface(t *testing.T) {
	// Test that the interface can be implemented
	var runnable ProgressiveRunnable

	// This should compile - we're just testing the interface
	assert.Nil(t, runnable)
}

// MockProgressiveRunnable is a simple implementation for testing.
type MockProgressiveRunnable struct {
	*BaseCommand
}

func (m *MockProgressiveRunnable) Run(ctx context.Context) Results {
	return Results{&Result{
		Label:  m.Label,
		Status: ResultStatusSuccess,
	}}
}

func (m *MockProgressiveRunnable) RunWithProgress(ctx context.Context, reporter progress.Reporter) Results {
	// Report start
	reporter.Report(progress.Event{
		CommandPath: []string{m.Label},
		Type:        progress.EventStarted,
		Message:     "Starting " + m.Label,
	})

	// Report some progress
	reporter.Report(progress.Event{
		CommandPath: []string{m.Label},
		Type:        progress.EventProgress,
		Message:     "Processing " + m.Label,
	})

	// Report completion
	reporter.Report(progress.Event{
		CommandPath: []string{m.Label},
		Type:        progress.EventCompleted,
		Message:     "Completed " + m.Label,
	})

	return Results{&Result{
		Label:  m.Label,
		Status: ResultStatusSuccess,
	}}
}

func TestMockProgressiveRunnable(t *testing.T) {
	ctx := context.Background()

	reporter := progress.NewChannelReporter(ctx, 10)
	defer reporter.Close()

	mock := &MockProgressiveRunnable{
		BaseCommand: &BaseCommand{
			Label: "test-command",
		},
	}

	// Verify it implements both interfaces
	var _ Runnable = mock

	var _ ProgressiveRunnable = mock

	// Test RunWithProgress
	results := mock.RunWithProgress(ctx, reporter)
	require.Len(t, results, 1)
	assert.Equal(t, "test-command", results[0].Label)
	assert.Equal(t, ResultStatusSuccess, results[0].Status)

	// Verify events were sent
	events := make([]progress.Event, 0, 3)

OuterLoop:
	for len(events) < 3 {
		select {
		case event := <-reporter.Events():
			events = append(events, event)
		default:
			break OuterLoop
		}
	}

	require.Len(t, events, 3)
	assert.Equal(t, progress.EventStarted, events[0].Type)
	assert.Equal(t, progress.EventProgress, events[1].Type)
	assert.Equal(t, progress.EventCompleted, events[2].Type)
}
