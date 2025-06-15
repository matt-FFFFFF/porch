// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package tui

import (
	"context"
	"testing"
	"time"

	"github.com/matt-FFFFFF/porch/internal/progress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommandNode(t *testing.T) {
	path := []string{"build", "test"}
	name := "unit-tests"

	node := NewCommandNode(path, name)

	require.NotNil(t, node)
	assert.Equal(t, path, node.Path)
	assert.Equal(t, name, node.Name)
	assert.Equal(t, StatusPending, node.Status)
	assert.Nil(t, node.StartTime)
	assert.Nil(t, node.EndTime)
	assert.Empty(t, node.LastOutput)
	assert.Empty(t, node.ErrorMsg)
	assert.Empty(t, node.Children)
}

func TestCommandNode_UpdateStatus(t *testing.T) {
	node := NewCommandNode([]string{"test"}, "command")

	// Test setting to running
	node.UpdateStatus(StatusRunning)
	status, _, _, _, startTime, endTime := node.GetDisplayInfo()
	assert.Equal(t, StatusRunning, status)
	assert.NotNil(t, startTime)
	assert.Nil(t, endTime)

	// Test setting to success
	node.UpdateStatus(StatusSuccess)
	status, _, _, _, startTime, endTime = node.GetDisplayInfo()
	assert.Equal(t, StatusSuccess, status)
	assert.NotNil(t, startTime)
	assert.NotNil(t, endTime)
}

func TestCommandNode_UpdateOutput(t *testing.T) {
	node := NewCommandNode([]string{"test"}, "command")

	// Test single line output
	node.UpdateOutput("Single line output")
	_, _, output, _, _, _ := node.GetDisplayInfo()
	assert.Equal(t, "Single line output", output)

	// Test multi-line output (should keep only last line)
	node.UpdateOutput("Line 1\nLine 2\nLine 3")
	_, _, output, _, _, _ = node.GetDisplayInfo()
	assert.Equal(t, "Line 3", output)

	// Test with trailing whitespace
	node.UpdateOutput("   Trimmed line   \n")
	_, _, output, _, _, _ = node.GetDisplayInfo()
	assert.Equal(t, "Trimmed line", output)
}

func TestCommandNode_UpdateError(t *testing.T) {
	node := NewCommandNode([]string{"test"}, "command")

	errorMsg := "Something went wrong"
	node.UpdateError(errorMsg)

	_, _, _, errMsg, _, _ := node.GetDisplayInfo()
	assert.Equal(t, errorMsg, errMsg)
}

func TestModel_GetOrCreateNode(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test creating new node
	path := []string{"build", "test"}
	name := "unit-tests"
	node := model.getOrCreateNode(path, name)

	require.NotNil(t, node)
	assert.Equal(t, path, node.Path)
	assert.Equal(t, name, node.Name)

	// Test getting existing node
	existingNode := model.getOrCreateNode(path, name)
	assert.Same(t, node, existingNode)

	// Verify it's in the nodeMap
	pathKey := pathToString(path)
	assert.Contains(t, model.nodeMap, pathKey)
	assert.Same(t, node, model.nodeMap[pathKey])
}

func TestModel_ProcessProgressEvent(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	commandPath := []string{"build", "test"}

	// Test EventStarted
	event := progress.Event{
		CommandPath: commandPath,
		Type:        progress.EventStarted,
		Message:     "Starting test",
		Timestamp:   time.Now(),
	}

	model.processProgressEvent(event)

	pathKey := pathToString(commandPath)
	node, exists := model.nodeMap[pathKey]
	require.True(t, exists)

	status, _, _, _, _, _ := node.GetDisplayInfo()
	assert.Equal(t, StatusRunning, status)

	// Test EventCompleted
	event = progress.Event{
		CommandPath: commandPath,
		Type:        progress.EventCompleted,
		Message:     "Test completed",
		Timestamp:   time.Now(),
	}

	model.processProgressEvent(event)

	status, _, _, _, _, _ = node.GetDisplayInfo()
	assert.Equal(t, StatusSuccess, status)

	// Test EventFailed
	event = progress.Event{
		CommandPath: commandPath,
		Type:        progress.EventFailed,
		Message:     "Test failed",
		Timestamp:   time.Now(),
		Data: progress.EventData{
			Error: assert.AnError,
		},
	}

	model.processProgressEvent(event)

	status, _, _, errMsg, _, _ := node.GetDisplayInfo()
	assert.Equal(t, StatusFailed, status)
	assert.Contains(t, errMsg, "assert.AnError")
}

func TestTUIReporter(t *testing.T) {
	// This is a basic test since we can't easily test the full bubbletea integration
	reporter := &Reporter{}

	// Test that reporting on a nil program doesn't panic
	event := progress.Event{
		CommandPath: []string{"test"},
		Type:        progress.EventStarted,
		Message:     "Test message",
		Timestamp:   time.Now(),
	}

	assert.NotPanics(t, func() {
		reporter.Report(event)
	})

	// Test close
	assert.NotPanics(t, func() {
		reporter.Close()
	})

	// Test that reporting after close doesn't panic
	assert.NotPanics(t, func() {
		reporter.Report(event)
	})
}

func TestPathToString(t *testing.T) {
	tests := []struct {
		name     string
		path     []string
		expected string
	}{
		{
			name:     "empty path",
			path:     []string{},
			expected: "",
		},
		{
			name:     "single element",
			path:     []string{"build"},
			expected: "build",
		},
		{
			name:     "multiple elements",
			path:     []string{"build", "test", "unit"},
			expected: "build/test/unit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pathToString(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
