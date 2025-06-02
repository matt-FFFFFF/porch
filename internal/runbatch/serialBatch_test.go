// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

type fakeCmd struct {
	label    string
	exitCode int
	err      error
}

func (f *fakeCmd) Run(_ context.Context) Results {
	return Results{&Result{
		Label:    f.label,
		ExitCode: f.exitCode,
		Error:    f.err,
	}}
}

// GetParent returns the parent batch of the command.
func (f *fakeCmd) GetParent() Runnable {
	// No-op for the fake command
	return nil
}

// SetParent sets the parent batch for the command.
func (f *fakeCmd) SetParent(_ Runnable) {
	// No-op for the fake command
}

// GetLabel returns the label of the batch.
func (c *fakeCmd) GetLabel() string {
	return c.label
}

func (f *fakeCmd) SetCwd(_ string, _ bool) {
	// No-op for the fake command
}

// InheritEnv sets the environment variables for the batch.
func (f *fakeCmd) InheritEnv(_ map[string]string) {
	// No-op for the fake command
}

func (f *fakeCmd) ShouldRun(_ RunState) bool {
	// Always run for the fake command
	return true
}

func TestSerialBatchRun_AllSuccess(t *testing.T) {
	defer goleak.VerifyNone(t)

	batch := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "batch1",
		},
		Commands: []Runnable{
			&fakeCmd{label: "cmd1", exitCode: 0},
			&fakeCmd{label: "cmd2", exitCode: 0},
		},
	}
	results := batch.Run(context.Background())
	assert.Len(t, results, 1)
	res := results[0]
	assert.Equal(t, 0, res.ExitCode)
	require.NoError(t, res.Error)
	assert.Len(t, res.Children, 2)
}

func TestSerialBatchRun_OneFailure(t *testing.T) {
	defer goleak.VerifyNone(t)

	batch := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "batch2",
		},
		Commands: []Runnable{
			&fakeCmd{label: "cmd1", exitCode: 0},
			&fakeCmd{label: "cmd2", exitCode: 1, err: os.ErrPermission},
		},
	}
	results := batch.Run(context.Background())
	assert.Len(t, results, 1)
	res := results[0]
	assert.NotEqual(t, 0, res.ExitCode)
	require.Error(t, res.Error)
	assert.Len(t, res.Children, 2)
}

func TestSerialBatchRun_NestedBatch(t *testing.T) {
	defer goleak.VerifyNone(t)

	childBatch := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "child",
		},
		Commands: []Runnable{
			&fakeCmd{label: "cmdA", exitCode: 0},
			&fakeCmd{label: "cmdB", exitCode: 1, err: os.ErrNotExist},
		},
	}
	batch := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "parent",
		},
		Commands: []Runnable{
			childBatch,
			&fakeCmd{label: "cmdC", exitCode: 0},
		},
	}
	childBatch.SetParent(batch)
	results := batch.Run(context.Background())
	assert.Len(t, results, 1)
	res := results[0]
	assert.Equal(t, -1, res.ExitCode)
	assert.ErrorIs(t, res.Error, ErrResultChildrenHasError)
}
