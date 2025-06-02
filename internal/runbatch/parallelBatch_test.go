// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

type fakeParallelCmd struct {
	label    string
	delay    time.Duration
	exitCode int
	err      error
}

// Run implements the Runnable interface for fakeParallelCmd.
func (f *fakeParallelCmd) Run(_ context.Context) Results {
	time.Sleep(f.delay)

	return Results{&Result{
		Label:    f.label,
		ExitCode: f.exitCode,
		Error:    f.err,
	}}
}

// GetLabel returns the label of the batch.
func (c *fakeParallelCmd) GetLabel() string {
	return c.label
}

// GetParent returns the parent for this command.
func (f *fakeParallelCmd) GetParent() Runnable {
	return nil // No parent for the fake command
}

// SetParent sets the parent batch for the command.
func (f *fakeParallelCmd) SetParent(_ Runnable) {
	// No-op for the fake command
}

// SetCwd implements the Runnable interface for fakeParallelCmd.
func (f *fakeParallelCmd) SetCwd(_ string, _ bool) {
	// No-op for the fake command
}

func (f *fakeParallelCmd) InheritEnv(_ map[string]string) {
	// No-op for the fake command
}

func (f *fakeParallelCmd) ShouldRun(_ RunState) bool {
	// Always run for the fake command
	return true
}

func TestParallelBatchRun_AllSuccess(t *testing.T) {
	defer goleak.VerifyNone(t)

	batch := &ParallelBatch{
		BaseCommand: &BaseCommand{
			Label: "parallel-batch-success",
		},
		Commands: []Runnable{
			&fakeParallelCmd{label: "cmd1", delay: 10 * time.Millisecond, exitCode: 0},
			&fakeParallelCmd{label: "cmd2", delay: 20 * time.Millisecond, exitCode: 0},
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
	defer goleak.VerifyNone(t)

	batch := &ParallelBatch{
		BaseCommand: &BaseCommand{
			Label: "parallel-batch-fail",
		},
		Commands: []Runnable{
			&fakeParallelCmd{label: "cmd1", delay: 10 * time.Millisecond, exitCode: 0},
			&fakeParallelCmd{label: "cmd2", delay: 10 * time.Millisecond, exitCode: 1, err: os.ErrPermission},
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
	defer goleak.VerifyNone(t)

	batch := &ParallelBatch{
		BaseCommand: &BaseCommand{
			Label: "parallel-batch-parallelism",
		},
		Commands: []Runnable{
			&fakeParallelCmd{label: "cmd1", delay: 100 * time.Millisecond, exitCode: 0},
			&fakeParallelCmd{label: "cmd2", delay: 100 * time.Millisecond, exitCode: 0},
		},
	}
	ctx := context.Background()
	start := time.Now()
	_ = batch.Run(ctx)
	duration := time.Since(start)
	assert.Less(t, duration, 180*time.Millisecond, "expected parallel execution to be faster than serial")
}
