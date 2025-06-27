// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeParallelCmd struct {
	*BaseCommand
	delay    time.Duration
	exitCode int
	err      error
}

// Run implements the Runnable interface for fakeParallelCmd.
func (f *fakeParallelCmd) Run(_ context.Context) Results {
	time.Sleep(f.delay)

	return Results{&Result{
		Label:    f.Label,
		ExitCode: f.exitCode,
		Error:    f.err,
	}}
}

func TestParallelBatchRun_AllSuccess(t *testing.T) {
	batch := &ParallelBatch{
		BaseCommand: &BaseCommand{
			Label: "parallel-batch-success",
		},
		Commands: []Runnable{
			&fakeParallelCmd{BaseCommand: &BaseCommand{Label: "cmd1"}, delay: 10 * time.Millisecond, exitCode: 0},
			&fakeParallelCmd{BaseCommand: &BaseCommand{Label: "cmd2"}, delay: 20 * time.Millisecond, exitCode: 0},
		},
	}
	ctx := context.Background()
	results := batch.Run(ctx)
	assert.Len(t, results, 1)
	require.NoError(t, results[0].Error, "expected no error")
	assert.Len(t, results[0].Children, 2, "expected 2 child results")

	for _, res := range results[0].Children {
		assert.Equal(t, 0, res.ExitCode)
		assert.NoError(t, res.Error)
	}
}

func TestParallelBatchRun_OneFailure(t *testing.T) {
	batch := &ParallelBatch{
		BaseCommand: &BaseCommand{
			Label: "parallel-batch-fail",
		},
		Commands: []Runnable{
			&fakeParallelCmd{
				BaseCommand: &BaseCommand{Label: "cmd1"},
				delay:       10 * time.Millisecond,
				exitCode:    0,
			},
			&fakeParallelCmd{
				BaseCommand: &BaseCommand{Label: "cmd2"},
				delay:       10 * time.Millisecond,
				exitCode:    1,
				err:         os.ErrPermission,
			},
		},
	}
	ctx := context.Background()
	results := batch.Run(ctx)
	assert.Len(t, results, 1)

	foundFail := false

	for _, res := range results[0].Children {
		if res.ExitCode != 0 {
			foundFail = true

			require.Error(t, res.Error)
		}
	}

	assert.True(t, foundFail, "expected at least one failure")
}

func TestParallelBatchRun_Parallelism(t *testing.T) {
	batch := &ParallelBatch{
		BaseCommand: &BaseCommand{
			Label: "parallel-batch-parallelism",
		},
		Commands: []Runnable{
			&fakeParallelCmd{BaseCommand: &BaseCommand{Label: "cmd1"}, delay: 100 * time.Millisecond, exitCode: 0},
			&fakeParallelCmd{BaseCommand: &BaseCommand{Label: "cmd2"}, delay: 100 * time.Millisecond, exitCode: 0},
		},
	}
	ctx := context.Background()
	start := time.Now()
	_ = batch.Run(ctx)
	duration := time.Since(start)
	assert.Less(t, duration, 180*time.Millisecond, "expected parallel execution to be faster than serial")
}

func TestParallelBatch_InheritsCwdFromSerialPredecessors(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create test directory structure
	testDir := filepath.Join(tempDir, "test")
	require.NoError(t, os.MkdirAll(testDir, 0o755))

	// Create a parallel batch that has been updated with a new cwd (simulating what SerialBatch does)
	parallelBatch := &ParallelBatch{
		BaseCommand: &BaseCommand{
			Label: "parallel-batch",
			Cwd:   testDir, // This simulates the cwd being updated by SerialBatch
		},
		Commands: []Runnable{
			&MockCommand{
				BaseCommand: &BaseCommand{
					Label: "cmd1",
					Cwd:   "", // Empty cwd should inherit from parallel batch
				},
			},
			&MockCommand{
				BaseCommand: &BaseCommand{
					Label: "cmd2",
					Cwd:   "./relative", // Existing relative path should be preserved
				},
			},
		},
	}

	// Run the parallel batch
	ctx := context.Background()
	results := parallelBatch.Run(ctx)

	// Verify results
	assert.NotNil(t, results)
	assert.Len(t, results, 1)
	assert.Equal(t, ResultStatusSuccess, results[0].Status)

	// Check that the parallel batch commands received the correct cwd after SetCwd was called
	cmd1 := parallelBatch.Commands[0].(*MockCommand)
	assert.Equal(t, testDir, cmd1.Cwd) // Empty cwd should inherit from parallel batch

	cmd2 := parallelBatch.Commands[1].(*MockCommand)
	// With overwrite=false, existing relative cwd should be resolved against batch cwd
	expectedPath := filepath.Join(testDir, "relative")
	assert.Equal(t, expectedPath, cmd2.Cwd)
}

func TestParallelBatch_CommandsDoNotInheritCwdFromSiblings(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create a parallel batch with commands that might change cwd
	parallelBatch := &ParallelBatch{
		BaseCommand: &BaseCommand{
			Label: "parallel-batch",
			Cwd:   tempDir,
		},
		Commands: []Runnable{
			&MockCommand{
				BaseCommand: &BaseCommand{
					Label: "cmd1",
					Cwd:   "",
				},
			},
			&MockCommand{
				BaseCommand: &BaseCommand{
					Label: "cmd2",
					Cwd:   "",
				},
			},
		},
	}

	// Run the parallel batch
	ctx := context.Background()
	results := parallelBatch.Run(ctx)

	// Verify results
	assert.NotNil(t, results)
	assert.Len(t, results, 1)
	assert.Equal(t, ResultStatusSuccess, results[0].Status)

	// Both commands should have the same initial cwd, not affected by each other
	cmd1 := parallelBatch.Commands[0].(*MockCommand)
	cmd2 := parallelBatch.Commands[1].(*MockCommand)

	assert.Equal(t, tempDir, cmd1.Cwd)
	assert.Equal(t, tempDir, cmd2.Cwd)
}

// MockCommand is a test implementation of Runnable.
type MockCommand struct {
	*BaseCommand
}

func (m *MockCommand) Run(ctx context.Context) Results {
	result := &Result{
		Label:    m.Label,
		ExitCode: 0,
		Status:   ResultStatusSuccess,
	}

	return Results{result}
}
