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
	defer goleak.VerifyNone(t)

	batch := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "batch1",
		},
		Commands: []Runnable{
			&fakeCmd{
				BaseCommand: &BaseCommand{
					Label: "cmd1",
				},
				exitCode: 0,
			},
			&fakeCmd{
				BaseCommand: &BaseCommand{
					Label: "cmd2",
				},
				exitCode: 0,
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
	defer goleak.VerifyNone(t)

	batch := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "batch2",
		},
		Commands: []Runnable{
			&fakeCmd{
				BaseCommand: &BaseCommand{
					Label: "cmd1",
				},
				exitCode: 0,
			},
			&fakeCmd{
				BaseCommand: &BaseCommand{
					Label: "cmd2",
				},
				exitCode: 1,
				err:      os.ErrPermission,
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
	defer goleak.VerifyNone(t)

	childBatch := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "child",
		},
		Commands: []Runnable{
			&fakeCmd{
				BaseCommand: &BaseCommand{
					Label: "cmdA",
				},
				exitCode: 0,
			},
			&fakeCmd{
				BaseCommand: &BaseCommand{
					Label: "cmdB",
				},
				exitCode: 1,
				err:      os.ErrNotExist,
			},
		},
	}
	batch := &SerialBatch{
		BaseCommand: &BaseCommand{
			Label: "parent",
		},
		Commands: []Runnable{
			childBatch,
			&fakeCmd{
				BaseCommand: &BaseCommand{
					Label: "cmdC",
				},
				exitCode: 0,
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
