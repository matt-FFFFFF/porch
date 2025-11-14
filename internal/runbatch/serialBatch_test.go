// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCmd struct {
	*BaseCommand
	exitCode int
	err      error
}

func (f *fakeCmd) Run(_ context.Context) Results {
	status := ResultStatusSuccess
	if f.err != nil || f.exitCode != 0 {
		status = ResultStatusError
	}

	return Results{&Result{
		Label:    f.Label,
		ExitCode: f.exitCode,
		Error:    f.err,
		Status:   status,
	}}
}

func TestSerialBatchRun_AllSuccess(t *testing.T) {
	batch := &SerialBatch{
		BaseCommand: NewBaseCommand("batch1", t.TempDir(), RunOnAlways, nil, nil),
		Commands: []Runnable{
			&fakeCmd{
				BaseCommand: NewBaseCommand("cmd1", t.TempDir(), RunOnAlways, nil, nil),
				exitCode:    0,
			},
			&fakeCmd{
				BaseCommand: NewBaseCommand("cmd2", t.TempDir(), RunOnAlways, nil, nil),
				exitCode:    0,
			},
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
	batch := &SerialBatch{
		BaseCommand: NewBaseCommand("batch2", t.TempDir(), RunOnAlways, nil, nil),
		Commands: []Runnable{
			&fakeCmd{
				BaseCommand: NewBaseCommand("cmd1", t.TempDir(), RunOnAlways, nil, nil),
				exitCode:    0,
			},
			&fakeCmd{
				BaseCommand: NewBaseCommand("cmd2", t.TempDir(), RunOnAlways, nil, nil),
				exitCode:    1,
				err:         os.ErrPermission,
			},
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
	childBatch := &SerialBatch{
		BaseCommand: NewBaseCommand("child", t.TempDir(), RunOnAlways, nil, nil),
		Commands: []Runnable{
			&fakeCmd{
				BaseCommand: NewBaseCommand("cmdA", t.TempDir(), RunOnAlways, nil, nil),
				exitCode:    0,
			},
			&fakeCmd{
				BaseCommand: NewBaseCommand("cmdB", t.TempDir(), RunOnAlways, nil, nil),
				exitCode:    1,
				err:         os.ErrNotExist,
			},
		},
	}
	batch := &SerialBatch{
		BaseCommand: NewBaseCommand("parent", t.TempDir(), RunOnAlways, nil, nil),
		Commands: []Runnable{
			childBatch,
			&fakeCmd{
				BaseCommand: NewBaseCommand("cmdC", t.TempDir(), RunOnAlways, nil, nil),
				exitCode:    0,
			},
		},
	}
	childBatch.SetParent(batch)
	results := batch.Run(context.Background())
	assert.Len(t, results, 1)
	res := results[0]
	assert.Equal(t, -1, res.ExitCode)
	require.ErrorIs(t, res.Error, ErrResultChildrenHasError)
}
