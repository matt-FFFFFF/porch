package runbatch

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// cwdCapturingCmd is a command that captures the cwd it was run with
type cwdCapturingCmd struct {
	label    string
	exitCode int
	err      error
	cwd      string
	runWith  string
	newCwd   string // returned newCwd
}

func (c *cwdCapturingCmd) Run(_ context.Context, _ <-chan os.Signal) Results {
	c.runWith = c.cwd // capture what cwd was used when running
	return Results{&Result{
		Label:    c.label,
		ExitCode: c.exitCode,
		Error:    c.err,
		newCwd:   c.newCwd,
	}}
}

func (c *cwdCapturingCmd) GetLabel() string {
	return c.label
}

func (c *cwdCapturingCmd) SetCwd(cwd string) {
	c.cwd = cwd
}

// TestSerialBatchCwdPropagation tests that when a command changes its working directory,
// subsequent commands in the batch use the new working directory.
func TestSerialBatchCwdPropagation(t *testing.T) {
	// Setup commands
	cmd1 := &cwdCapturingCmd{label: "cmd1", exitCode: 0, cwd: "/initial/path", newCwd: "/new/path"}
	cmd2 := &cwdCapturingCmd{label: "cmd2", exitCode: 0, cwd: "/initial/path"}
	cmd3 := &cwdCapturingCmd{label: "cmd3", exitCode: 0, cwd: "/initial/path"}

	batch := &SerialBatch{
		Label:    "batch_with_cwd_changes",
		Commands: []Runnable{cmd1, cmd2, cmd3},
	}

	// Initial setup - all commands should have the initial path
	assert.Equal(t, "/initial/path", cmd1.cwd)
	assert.Equal(t, "/initial/path", cmd2.cwd)
	assert.Equal(t, "/initial/path", cmd3.cwd)

	// Run the batch
	results := batch.Run(context.Background(), nil)

	// Verify results
	assert.Len(t, results, 1)
	assert.Equal(t, 0, results[0].ExitCode)
	assert.NoError(t, results[0].Error)
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
	cmd1 := &cwdCapturingCmd{label: "cmd1", exitCode: 0, cwd: "/initial/path", newCwd: "/path/1"}
	cmd2 := &cwdCapturingCmd{label: "cmd2", exitCode: 0, cwd: "/initial/path", newCwd: "/path/2"}
	cmd3 := &cwdCapturingCmd{label: "cmd3", exitCode: 0, cwd: "/initial/path"}

	batch := &SerialBatch{
		Label:    "batch_with_multiple_cwd_changes",
		Commands: []Runnable{cmd1, cmd2, cmd3},
	}

	// Run the batch
	_ = batch.Run(context.Background(), nil)

	// Verify the last command picked up the most recent cwd change
	assert.Equal(t, "/initial/path", cmd1.runWith)
	assert.Equal(t, "/path/1", cmd2.runWith)
	assert.Equal(t, "/path/2", cmd3.runWith)
}

// TestSerialBatchCwdNoChange tests that when no command changes the working directory,
// all commands run with their original cwd.
func TestSerialBatchCwdNoChange(t *testing.T) {
	// Setup commands
	cmd1 := &cwdCapturingCmd{label: "cmd1", exitCode: 0, cwd: "/initial/path"}
	cmd2 := &cwdCapturingCmd{label: "cmd2", exitCode: 0, cwd: "/initial/path"}
	cmd3 := &cwdCapturingCmd{label: "cmd3", exitCode: 0, cwd: "/initial/path"}

	batch := &SerialBatch{
		Label:    "batch_with_no_cwd_changes",
		Commands: []Runnable{cmd1, cmd2, cmd3},
	}

	// Run the batch
	_ = batch.Run(context.Background(), nil)

	// All commands should have run with their initial paths
	assert.Equal(t, "/initial/path", cmd1.runWith)
	assert.Equal(t, "/initial/path", cmd2.runWith)
	assert.Equal(t, "/initial/path", cmd3.runWith)
}

// TestSerialBatchCwdErrorHandling tests that when a command returns multiple results or has an error,
// the cwd change is ignored.
func TestSerialBatchCwdErrorHandling(t *testing.T) {
	// A command that returns multiple results
	multiResultCmd := &customResultsCmd{
		results: Results{
			{Label: "sub1", ExitCode: 0, newCwd: "/should/be/ignored"},
			{Label: "sub2", ExitCode: 0},
		},
	}

	cmd2 := &cwdCapturingCmd{label: "cmd2", exitCode: 0, cwd: "/initial/path"}

	batch := &SerialBatch{
		Label:    "batch_with_multi_results",
		Commands: []Runnable{multiResultCmd, cmd2},
	}

	// Run the batch
	_ = batch.Run(context.Background(), nil)

	// cmd2 should not have picked up any cwd changes
	assert.Equal(t, "/initial/path", cmd2.runWith)

	// Test with error case
	errorCmd := &cwdCapturingCmd{
		label:    "error_cmd",
		exitCode: 1,
		err:      assert.AnError,
		cwd:      "/initial/path",
		newCwd:   "/should/be/ignored",
	}

	cmd3 := &cwdCapturingCmd{label: "cmd3", exitCode: 0, cwd: "/initial/path"}

	batch2 := &SerialBatch{
		Label:    "batch_with_error",
		Commands: []Runnable{errorCmd, cmd3},
	}

	// Run the batch
	_ = batch2.Run(context.Background(), nil)

	// cmd3 should not have picked up any cwd changes from the error command
	assert.Equal(t, "/initial/path", cmd3.runWith)
}

// customResultsCmd is a command that returns custom results
type customResultsCmd struct {
	results Results
	cwd     string
}

func (c *customResultsCmd) Run(_ context.Context, _ <-chan os.Signal) Results {
	return c.results
}

func (c *customResultsCmd) GetLabel() string {
	return "custom_results_cmd"
}

func (c *customResultsCmd) SetCwd(cwd string) {
	c.cwd = cwd
}

// TestSerialBatchCwdWithNestedBatches tests that cwd changes propagate through nested batches
func TestSerialBatchCwdWithNestedBatches(t *testing.T) {
	// Setup inner batch
	innerCmd1 := &cwdCapturingCmd{label: "inner_cmd1", exitCode: 0, cwd: "/initial/path"}
	innerCmd2 := &cwdCapturingCmd{label: "inner_cmd2", exitCode: 0, cwd: "/initial/path"}

	innerBatch := &SerialBatch{
		Label:    "inner_batch",
		Commands: []Runnable{innerCmd1, innerCmd2},
	}

	// Setup outer batch
	outerCmd1 := &cwdCapturingCmd{label: "outer_cmd1", exitCode: 0, cwd: "/initial/path", newCwd: "/new/path"}
	outerCmd2 := &cwdCapturingCmd{label: "outer_cmd2", exitCode: 0, cwd: "/initial/path"}

	outerBatch := &SerialBatch{
		Label:    "outer_batch",
		Commands: []Runnable{outerCmd1, innerBatch, outerCmd2},
	}

	// Run the outer batch
	outerBatch.Run(context.Background(), nil)

	// Check cwd propagation
	assert.Equal(t, "/initial/path", outerCmd1.runWith)
	assert.Equal(t, "/new/path", innerCmd1.runWith)
	assert.Equal(t, "/new/path", innerCmd2.runWith)
	assert.Equal(t, "/new/path", outerCmd2.runWith)
}
