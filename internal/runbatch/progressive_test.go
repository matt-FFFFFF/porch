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

func TestProgressContext(t *testing.T) {
	ctx := context.Background()
	reporter := progress.NewNullReporter()
	path := []string{"test", "command"}

	progressCtx := NewProgressContext(ctx, reporter, path)

	assert.Equal(t, reporter, progressCtx.Reporter)
	assert.Equal(t, path, progressCtx.CommandPath)
}

func TestProgressContext_Child(t *testing.T) {
	ctx := context.Background()
	reporter := progress.NewNullReporter()
	parentPath := []string{"parent"}

	parentCtx := NewProgressContext(ctx, reporter, parentPath)
	childCtx := parentCtx.Child("child")

	expectedPath := []string{"parent", "child"}
	assert.Equal(t, expectedPath, childCtx.CommandPath)
	assert.Equal(t, reporter, childCtx.Reporter)
	assert.Equal(t, ctx, childCtx.Context)
}

func TestProgressContext_DeepNesting(t *testing.T) {
	ctx := context.Background()
	reporter := progress.NewNullReporter()

	// Test deep nesting
	progressCtx := NewProgressContext(ctx, reporter, []string{"root"})

	for i := 0; i < 10; i++ {
		progressCtx = progressCtx.Child("level" + string(rune('0'+i)))
	}

	assert.Len(t, progressCtx.CommandPath, 11) // root + 10 levels
	assert.Equal(t, "root", progressCtx.CommandPath[0])
	assert.Equal(t, "level9", progressCtx.CommandPath[10])
}

func TestProgressiveRunnableInterface(t *testing.T) {
	// Test that the interface can be implemented
	var runnable ProgressiveRunnable

	// This should compile - we're just testing the interface
	assert.Nil(t, runnable)
}

// MockProgressiveRunnable is a simple implementation for testing
type MockProgressiveRunnable struct {
	*BaseCommand
	events []progress.ProgressEvent
}

func (m *MockProgressiveRunnable) Run(ctx context.Context) Results {
	return Results{&Result{
		Label:  m.Label,
		Status: ResultStatusSuccess,
	}}
}

func (m *MockProgressiveRunnable) RunWithProgress(ctx context.Context, reporter progress.ProgressReporter) Results {
	// Report start
	reporter.Report(progress.ProgressEvent{
		CommandPath: []string{m.Label},
		Type:        progress.EventStarted,
		Message:     "Starting " + m.Label,
	})

	// Report some progress
	reporter.Report(progress.ProgressEvent{
		CommandPath: []string{m.Label},
		Type:        progress.EventProgress,
		Message:     "Processing " + m.Label,
	})

	// Report completion
	reporter.Report(progress.ProgressEvent{
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
	events := make([]progress.ProgressEvent, 0, 3)

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
