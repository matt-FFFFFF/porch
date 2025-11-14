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
		BaseCommand: NewBaseCommand("parallel-batch-success", t.TempDir(), RunOnAlways, nil, nil),
		Commands: []Runnable{
			&fakeParallelCmd{BaseCommand: NewBaseCommand("cmd1", t.TempDir(), RunOnAlways, nil, nil), delay: 10 * time.Millisecond, exitCode: 0},
			&fakeParallelCmd{BaseCommand: NewBaseCommand("cmd2", t.TempDir(), RunOnAlways, nil, nil), delay: 20 * time.Millisecond, exitCode: 0},
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
		BaseCommand: NewBaseCommand("parallel-batch-fail", t.TempDir(), RunOnAlways, nil, nil),
		Commands: []Runnable{
			&fakeParallelCmd{
				BaseCommand: NewBaseCommand("cmd1", t.TempDir(), RunOnAlways, nil, nil),
				delay:       10 * time.Millisecond,
				exitCode:    0,
			},
			&fakeParallelCmd{
				BaseCommand: NewBaseCommand("cmd2", t.TempDir(), RunOnAlways, nil, nil),
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
		BaseCommand: NewBaseCommand("parallel-batch-parallelism", t.TempDir(), RunOnAlways, nil, nil),
		Commands: []Runnable{
			&fakeParallelCmd{BaseCommand: NewBaseCommand("cmd1", t.TempDir(), RunOnAlways, nil, nil), delay: 100 * time.Millisecond, exitCode: 0},
			&fakeParallelCmd{BaseCommand: NewBaseCommand("cmd2", t.TempDir(), RunOnAlways, nil, nil), delay: 100 * time.Millisecond, exitCode: 0},
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

	pb := &ParallelBatch{
		BaseCommand: NewBaseCommand("parallel-batch", testDir, RunOnAlways, nil, nil),
	}
	// Create a parallel batch that has been updated with a new cwd (simulating what SerialBatch does)
	cmd1 := NewBaseCommand("cmd1", tempDir, RunOnAlways, nil, nil)
	cmd1.parent = pb
	cmd2 := NewBaseCommand("cmd2", filepath.Join(tempDir, "relative"), RunOnAlways, nil, nil)
	cmd2.parent = pb
	pb.Commands = []Runnable{
		&MockCommand{
			BaseCommand: cmd1, // Commands should start with absolute paths (as resolved at creation)
		},
		&MockCommand{
			BaseCommand: cmd2, // Relative paths should already be resolved to absolute
		},
	}

	// Run the parallel batch
	ctx := context.Background()
	results := pb.Run(ctx)

	// Verify results
	assert.NotNil(t, results)
	assert.Len(t, results, 1)
	assert.Equal(t, ResultStatusSuccess, results[0].Status)

	// Check that the parallel batch commands received the correct cwd after SetCwd was called
	mockCmd1 := pb.Commands[0].(*MockCommand)
	assert.Equal(t, tempDir, mockCmd1.GetCwd()) // Command cwd should prefer the testDir

	mockCmd2 := pb.Commands[1].(*MockCommand)
	// The relative path was already resolved to absolute, so SetCwd should preserve it
	expectedPath := filepath.Join(tempDir, "relative")
	assert.Equal(t, expectedPath, mockCmd2.GetCwd())
}

func TestParallelBatch_CommandsDoNotInheritCwdFromSiblings(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create a parallel batch with commands that might change cwd
	parallelBatch := &ParallelBatch{
		BaseCommand: NewBaseCommand("parallel-batch", tempDir, RunOnAlways, nil, nil),
	}
	pbCmd1 := NewBaseCommand("cmd1", tempDir, RunOnAlways, nil, nil)
	pbCmd1.parent = parallelBatch
	pbCmd2 := NewBaseCommand("cmd2", tempDir, RunOnAlways, nil, nil)
	pbCmd2.parent = parallelBatch
	parallelBatch.Commands = []Runnable{
		&MockCommand{
			BaseCommand: pbCmd1, // Commands should have absolute paths
		},
		&MockCommand{
			BaseCommand: pbCmd2, // Commands should have absolute paths
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
	mockCmd1 := parallelBatch.Commands[0].(*MockCommand)
	mockCmd2 := parallelBatch.Commands[1].(*MockCommand)

	assert.Equal(t, tempDir, mockCmd1.GetCwd())
	assert.Equal(t, tempDir, mockCmd2.GetCwd())
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
