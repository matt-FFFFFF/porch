// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cwdCapturingCmd is a command that captures the cwd it was run with.
type cwdCapturingCmd struct {
	*BaseCommand
	exitCode int
	err      error
	runWith  string
	newCwd   string // returned newCwd
	status   ResultStatus
}

func (c *cwdCapturingCmd) Run(_ context.Context) Results {
	c.runWith = c.GetCwd() // capture what cwd was used when running

	return Results{&Result{
		Label:    c.Label,
		ExitCode: c.exitCode,
		Error:    c.err,
		newCwd:   c.newCwd,
		Status:   c.status,
	}}
}

// TestSerialBatchCwdPropagation tests that when a command changes its working directory,
// subsequent commands in the batch use the new working directory.
func TestSerialBatchCwdPropagation(t *testing.T) {
	// Setup commands
	cmd1 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("cmd1", "", RunOnAlways, nil, nil),
		exitCode:    0,
		newCwd:      "/new/path",
		status:      ResultStatusSuccess,
	}
	cmd2 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("cmd2", "", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}
	cmd3 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("cmd3", "", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}

	batch := &SerialBatch{
		BaseCommand: NewBaseCommand("batch_with_cwd_changes", ".", RunOnAlways, nil, nil),
		Commands:    []Runnable{cmd1, cmd2, cmd3},
	}

	for _, cmd := range batch.Commands {
		cmd.SetParent(batch) // Set parent for proper context
	}

	// Initial setup - all commands should have the initial path
	assert.Equal(t, ".", cmd1.GetCwd())
	assert.Equal(t, ".", cmd2.GetCwd())
	assert.Equal(t, ".", cmd3.GetCwd())

	// Run the batch
	results := batch.Run(context.Background())

	// Verify results
	assert.Len(t, results, 1)
	assert.Equal(t, 0, results[0].ExitCode)
	require.NoError(t, results[0].Error)
	assert.Len(t, results[0].Children, 3)

	// First command ran with initial path
	assert.Equal(t, ".", cmd1.runWith)

	// Subsequent commands should have been updated to run with the new path
	assert.Equal(t, "/new/path", cmd2.runWith)
	assert.Equal(t, "/new/path", cmd3.runWith)
}

// TestSerialBatchCwdMultipleChanges tests that when multiple commands change their working
// directory, the latest change is always propagated to subsequent commands.
func TestSerialBatchCwdMultipleChanges(t *testing.T) {
	tmpDir := t.TempDir()
	// Setup commands
	// The first commadn sets an absolute path, which will override the batch's initial cwd.
	cmd1 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("cmd1", ".", RunOnAlways, nil, nil),
		exitCode:    0,
		newCwd:      "/path/1",
		status:      ResultStatusSuccess,
	}
	// The second command sets a relative path, which will be resolved against the first command's new cwd.
	cmd2 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("cmd2", "subdir", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}
	// The third command sets another absolute path. This will be ignored as the command now has an absolute path set.
	cmd3 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("cmd3", ".", RunOnAlways, nil, nil),
		exitCode:    0,
		newCwd:      "/path/2",
		status:      ResultStatusSuccess,
	}
	cmd4 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("cmd4", ".", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}

	batch := &SerialBatch{
		BaseCommand: NewBaseCommand("batch_with_multiple_cwd_changes", tmpDir, RunOnAlways, nil, nil),
		Commands:    []Runnable{cmd1, cmd2, cmd3, cmd4},
	}

	for _, cmd := range batch.Commands {
		cmd.SetParent(batch) // Set parent for proper context
	}

	// Run the batch
	_ = batch.Run(t.Context())

	// Verify the last command picked up the most recent cwd change
	assert.Equal(t, tmpDir, cmd1.runWith)
	assert.Equal(t, "/path/1/subdir", cmd2.runWith)
	assert.Equal(t, "/path/2", cmd4.runWith)
}

// TestSerialBatchCwdNoChange tests that when no command changes the working directory,
// all commands run with their original cwd.
func TestSerialBatchCwdNoChange(t *testing.T) {
	// Setup commands
	cmd1 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("cmd1", ".", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}
	cmd2 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("cmd2", ".", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}
	cmd3 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("cmd3", ".", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}

	batch := &SerialBatch{
		BaseCommand: NewBaseCommand("batch_with_no_cwd_changes", t.TempDir(), RunOnAlways, nil, nil),
		Commands:    []Runnable{cmd1, cmd2, cmd3},
	}

	// Run the batch
	_ = batch.Run(context.Background())

	// All commands should have run with their initial paths
	assert.Equal(t, ".", cmd1.runWith)
	assert.Equal(t, ".", cmd2.runWith)
	assert.Equal(t, ".", cmd3.runWith)
}

// TestSerialBatchCwdErrorHandling tests that when a command returns multiple results or has an error,
// the cwd change is ignored.
func TestSerialBatchCwdErrorHandling(t *testing.T) {
	// Test with error case
	errorCmd := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("error_cmd", ".", RunOnAlways, nil, nil),
		exitCode:    1,
		err:         assert.AnError,
		newCwd:      "/should/be/propagated",
		status:      ResultStatusError,
	}

	cmd3 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("cwd3", ".", RunOnAlways, nil, nil),
		exitCode:    0,
	}

	batch2 := &SerialBatch{
		BaseCommand: NewBaseCommand("batch_with_error", t.TempDir(), RunOnAlways, nil, nil),
		Commands:    []Runnable{errorCmd, cmd3},
	}
	for _, cmd := range batch2.Commands {
		cmd.SetParent(batch2) // Set parent for proper context
	}

	// Run the batch
	_ = batch2.Run(context.Background())

	// cmd3 should have picked up any cwd changes from the error command
	assert.Equal(t, "/should/be/propagated", cmd3.runWith)
}

// TestSerialBatchCwdWithNestedBatches tests that cwd changes propagate through nested batches.
func TestSerialBatchCwdWithNestedBatches(t *testing.T) {
	// Setup inner batch
	innerCmd1 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("inner_cmd1", "", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}
	innerCmd2 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("inner_cmd2", "", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}

	innerBatch := &SerialBatch{
		BaseCommand: NewBaseCommand("inner_batch", "", RunOnAlways, nil, nil),
		Commands:    []Runnable{innerCmd1, innerCmd2},
	}
	for _, cmd := range innerBatch.Commands {
		cmd.SetParent(innerBatch) // Set parent for proper context
	}

	// Setup outer batch
	outerCmd1 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("outer_cmd1", "", RunOnAlways, nil, nil),
		exitCode:    0,
		newCwd:      "/new/path",
		status:      ResultStatusSuccess,
	}
	outerCmd2 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("outer_cmd2", "", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}

	outerBatch := &SerialBatch{
		BaseCommand: NewBaseCommand("outer_batch", ".", RunOnAlways, nil, nil),
		Commands:    []Runnable{outerCmd1, innerBatch, outerCmd2},
	}
	for _, cmd := range outerBatch.Commands {
		cmd.SetParent(outerBatch) // Set parent for proper context
	}

	// Run the outer batch
	outerBatch.Run(context.Background())

	// Check cwd propagation
	assert.Equal(t, ".", outerCmd1.runWith)
	assert.Equal(t, "/new/path", innerCmd1.runWith)
	assert.Equal(t, "/new/path", innerCmd2.runWith)
	assert.Equal(t, "/new/path", outerCmd2.runWith)
}

// TestSerialBatchCwdWithNestedNestedBatches tests that cwd changes propagate through nested batches.
func TestSerialBatchCwdWithNestedNestedBatches(t *testing.T) {
	// Setup inner batch
	innerCmd1 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("inner_cmd1", "", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}
	innerCmd2 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("inner_cmd2", "", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}
	innerCmd3 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("inner_cmd3", "", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}
	innerCmd4 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("inner_cmd4", "", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}

	// Inner batch 2 has a relative cwd change
	innerBatch2 := &SerialBatch{
		BaseCommand: NewBaseCommand("inner_batch_2", "./new/path", RunOnAlways, nil, nil),
		Commands:    []Runnable{innerCmd3, innerCmd4},
	}

	for _, cmd := range innerBatch2.Commands {
		cmd.SetParent(innerBatch2) // Set parent for proper context
	}

	innerBatch1 := &SerialBatch{
		BaseCommand: NewBaseCommand("inner_batch_1", "", RunOnAlways, nil, nil),
		Commands:    []Runnable{innerCmd1, innerCmd2, innerBatch2},
	}

	for _, cmd := range innerBatch1.Commands {
		cmd.SetParent(innerBatch1) // Set parent for proper context
	}

	// Setup outer batch
	outerCmd1 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("outer_cmd1", "", RunOnAlways, nil, nil),
		exitCode:    0,
		newCwd:      "/new/path",
		status:      ResultStatusSuccess,
	}
	outerCmd2 := &cwdCapturingCmd{
		BaseCommand: NewBaseCommand("outer_cmd2", "", RunOnAlways, nil, nil),
		exitCode:    0,
		status:      ResultStatusSuccess,
	}

	outerBatch := &SerialBatch{
		BaseCommand: NewBaseCommand("outer_batch", ".", RunOnAlways, nil, nil),
		Commands:    []Runnable{outerCmd1, innerBatch1, outerCmd2},
	}
	for _, cmd := range outerBatch.Commands {
		cmd.SetParent(outerBatch) // Set parent for proper context
	}

	// Run the outer batch
	outerBatch.Run(t.Context())

	// Check cwd propagation
	assert.Equal(t, ".", outerCmd1.runWith)
	assert.Equal(t, "/new/path", innerCmd1.runWith)
	assert.Equal(t, "/new/path", innerCmd2.runWith)
	assert.Equal(t, "/new/path", outerCmd2.runWith)
	assert.Equal(t, "/new/path/new/path", innerCmd3.runWith)
	assert.Equal(t, "/new/path/new/path", innerCmd4.runWith)
}
