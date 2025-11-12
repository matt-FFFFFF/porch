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
	c.runWith = c.Cwd // capture what cwd was used when running

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
		BaseCommand: &BaseCommand{
			Label: "cmd1",
			Cwd:   "/initial/path",
		},
		exitCode: 0,
		newCwd:   "/new/path",
		status:   ResultStatusSuccess,
	}
	cmd2 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "cmd2",
			Cwd:   "/initial/path"},
		exitCode: 0,
		status:   ResultStatusSuccess,
	}
	cmd3 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "cmd3",
			Cwd:   "/initial/path"},
		exitCode: 0,
		status:   ResultStatusSuccess,
	}

	batch := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "batch_with_cwd_changes",
			Cwd:   "/",
		},
		Commands: []Runnable{cmd1, cmd2, cmd3},
	}

	for _, cmd := range batch.Commands {
		cmd.SetParent(batch) // Set parent for proper context
	}

	// Initial setup - all commands should have the initial path
	assert.Equal(t, "/initial/path", cmd1.Cwd)
	assert.Equal(t, "/initial/path", cmd2.Cwd)
	assert.Equal(t, "/initial/path", cmd3.Cwd)

	// Run the batch
	results := batch.Run(context.Background())

	// Verify results
	assert.Len(t, results, 1)
	assert.Equal(t, 0, results[0].ExitCode)
	require.NoError(t, results[0].Error)
	assert.Len(t, results[0].Children, 3)

	// First command ran with initial path
	assert.Equal(t, "/initial/path", cmd1.runWith)

	// Subsequent commands should have been updated to run with the new path
	assert.Equal(t, "/new/path", cmd2.runWith)
	assert.Equal(t, "/new/path", cmd3.runWith)
}

// TestSerialBatchCwdMultipleChanges tests that when multiple commands change their working
// directory, the latest change is always propagated to subsequent commands.
func TestSerialBatchCwdMultipleChanges(t *testing.T) {
	// Setup commands
	cmd1 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Cwd:   "/initial/path",
			Label: "cmd1",
		},
		exitCode: 0,
		newCwd:   "/path/1",
		status:   ResultStatusSuccess,
	}
	cmd2 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "cmd2",
			Cwd:   "/initial/path",
		},
		exitCode: 0,
		newCwd:   "/path/2",
		status:   ResultStatusSuccess,
	}
	cmd3 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "cmd3",
			Cwd:   "/initial/path",
		},
		exitCode: 0,
		status:   ResultStatusSuccess,
	}

	batch := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "batch_with_multiple_cwd_changes",
			Cwd:   t.TempDir(), // Use a temp dir for the batch
		},
		Commands: []Runnable{cmd1, cmd2, cmd3},
	}

	for _, cmd := range batch.Commands {
		cmd.SetParent(batch) // Set parent for proper context
	}

	// Run the batch
	_ = batch.Run(context.Background())

	// Verify the last command picked up the most recent cwd change
	assert.Equal(t, "/initial/path", cmd1.runWith)
	assert.Equal(t, "/path/1", cmd2.runWith)
	assert.Equal(t, "/path/2", cmd3.runWith)
}

// TestSerialBatchCwdNoChange tests that when no command changes the working directory,
// all commands run with their original cwd.
func TestSerialBatchCwdNoChange(t *testing.T) {
	// Setup commands
	cmd1 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "cmd1",
			Cwd:   "/initial/path",
		},
		exitCode: 0,
		status:   ResultStatusSuccess,
	}
	cmd2 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "cmd2",
			Cwd:   "/initial/path",
		},
		exitCode: 0,
		status:   ResultStatusSuccess,
	}
	cmd3 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "cmd3",
			Cwd:   "/initial/path",
		},
		exitCode: 0,
		status:   ResultStatusSuccess,
	}

	batch := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "batch_with_no_cwd_changes",
		},
		Commands: []Runnable{cmd1, cmd2, cmd3},
	}

	// Run the batch
	_ = batch.Run(context.Background())

	// All commands should have run with their initial paths
	assert.Equal(t, "/initial/path", cmd1.runWith)
	assert.Equal(t, "/initial/path", cmd2.runWith)
	assert.Equal(t, "/initial/path", cmd3.runWith)
}

// TestSerialBatchCwdErrorHandling tests that when a command returns multiple results or has an error,
// the cwd change is ignored.
func TestSerialBatchCwdErrorHandling(t *testing.T) {
	// Test with error case
	errorCmd := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "error_cmd",
			Cwd:   "/initial/path",
		},
		exitCode: 1,
		err:      assert.AnError,
		newCwd:   "/should/be/propagated",
		status:   ResultStatusError,
	}

	cmd3 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label:           "cwd3",
			Cwd:             "/initial/path",
			RunsOnCondition: RunOnAlways,
		},
		exitCode: 0,
	}

	batch2 := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "batch_with_error",
		},
		Commands: []Runnable{errorCmd, cmd3},
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
		BaseCommand: &BaseCommand{
			Label: "inner_cmd1",
			Cwd:   "/initial/path", // Commands should have absolute paths
		},
		exitCode: 0,
		status:   ResultStatusSuccess,
	}
	innerCmd2 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "inner_cmd2",
			Cwd:   "/initial/path", // Commands should have absolute paths
		},
		exitCode: 0,
		status:   ResultStatusSuccess,
	}

	innerBatch := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "inner_batch",
			Cwd:   "/initial/path",
		},
		Commands: []Runnable{innerCmd1, innerCmd2},
	}
	for _, cmd := range innerBatch.Commands {
		cmd.SetParent(innerBatch) // Set parent for proper context
	}

	// Setup outer batch
	outerCmd1 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "outer_cmd1",
			Cwd:   "/initial/path",
		},
		exitCode: 0,
		newCwd:   "/new/path",
		status:   ResultStatusSuccess,
	}
	outerCmd2 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "outer_cmd2",
			Cwd:   "/initial/path",
		},
		exitCode: 0,
		status:   ResultStatusSuccess,
	}

	outerBatch := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "outer_batch",
		},
		Commands: []Runnable{outerCmd1, innerBatch, outerCmd2},
	}
	for _, cmd := range outerBatch.Commands {
		cmd.SetParent(outerBatch) // Set parent for proper context
	}

	// Run the outer batch
	outerBatch.Run(context.Background())

	// Check cwd propagation
	assert.Equal(t, "/initial/path", outerCmd1.runWith)
	assert.Equal(t, "/new/path", innerCmd1.runWith)
	assert.Equal(t, "/new/path", innerCmd2.runWith)
	assert.Equal(t, "/new/path", outerCmd2.runWith)
}

// TestSerialBatchCwdWithNestedNestedBatches tests that cwd changes propagate through nested batches.
func TestSerialBatchCwdWithNestedNestedBatches(t *testing.T) {
	// Setup inner batch
	innerCmd1 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "inner_cmd1",
			Cwd:   "/initial/path", // Commands should have absolute paths
		},
		exitCode: 0,
		status:   ResultStatusSuccess,
	}
	innerCmd2 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "inner_cmd2",
			Cwd:   "/initial/path", // Commands should have absolute paths
		},
		exitCode: 0,
		status:   ResultStatusSuccess,
	}
	innerCmd3 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "inner_cmd3",
			Cwd:   "/initial/path", // Commands should have absolute paths
		},
		exitCode: 0,
		status:   ResultStatusSuccess,
	}
	innerCmd4 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "inner_cmd4",
			Cwd:   "/initial/path", // Commands should have absolute paths
		},
		exitCode: 0,
		status:   ResultStatusSuccess,
	}

	innerBatch2 := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label:  "inner_batch_2",
			Cwd:    "/initial/path",
			CwdRel: "./new/path",
		},
		Commands: []Runnable{innerCmd3, innerCmd4},
	}

	for _, cmd := range innerBatch2.Commands {
		cmd.SetParent(innerBatch2) // Set parent for proper context
	}

	innerBatch1 := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "inner_batch_1",
			Cwd:   "/initial/path",
		},
		Commands: []Runnable{innerCmd1, innerCmd2, innerBatch2},
	}

	for _, cmd := range innerBatch1.Commands {
		cmd.SetParent(innerBatch1) // Set parent for proper context
	}

	// Setup outer batch
	outerCmd1 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "outer_cmd1",
			Cwd:   "/initial/path",
		},
		exitCode: 0,
		newCwd:   "/new/path",
		status:   ResultStatusSuccess,
	}
	outerCmd2 := &cwdCapturingCmd{
		BaseCommand: &BaseCommand{
			Label: "outer_cmd2",
			Cwd:   "/initial/path",
		},
		exitCode: 0,
		status:   ResultStatusSuccess,
	}

	outerBatch := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "outer_batch",
		},
		Commands: []Runnable{outerCmd1, innerBatch1, outerCmd2},
	}
	for _, cmd := range outerBatch.Commands {
		cmd.SetParent(outerBatch) // Set parent for proper context
	}

	// Run the outer batch
	outerBatch.Run(context.Background())

	// Check cwd propagation
	assert.Equal(t, "/initial/path", outerCmd1.runWith)
	assert.Equal(t, "/new/path", innerCmd1.runWith)
	assert.Equal(t, "/new/path", innerCmd2.runWith)
	assert.Equal(t, "/new/path", outerCmd2.runWith)
	assert.Equal(t, "/new/path/new/path", innerCmd3.runWith)
	assert.Equal(t, "/new/path/new/path", innerCmd4.runWith)
}
