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

func (f *fakeCmd) SetCwd(_ string) {
	// No-op for the fake command
}

// InheritEnv sets the environment variables for the batch.
func (f *fakeCmd) InheritEnv(_ map[string]string) {
	// No-op for the fake command
}

func TestSerialBatchRun_AllSuccess(t *testing.T) {
	defer goleak.VerifyNone(t)

	batch := &SerialBatch{
		Label: "batch1",
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
		Label: "batch2",
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
		Label: "child",
		Commands: []Runnable{
			&fakeCmd{label: "cmdA", exitCode: 0},
			&fakeCmd{label: "cmdB", exitCode: 1, err: os.ErrNotExist},
		},
	}
	batch := &SerialBatch{
		Label: "parent",
		Commands: []Runnable{
			childBatch,
			&fakeCmd{label: "cmdC", exitCode: 0},
		},
	}
	results := batch.Run(context.Background())
	assert.Len(t, results, 1)
	res := results[0]
	assert.Equal(t, -1, res.ExitCode)
	assert.ErrorIs(t, res.Error, ErrResultChildrenHasError)
}
