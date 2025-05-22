package runbatch

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

type fakeParallelCmd struct {
	label    string
	delay    time.Duration
	exitCode int
	err      error
}

func (f *fakeParallelCmd) Run(ctx context.Context) Results {
	time.Sleep(f.delay)
	return Results{&Result{
		Label:    f.label,
		ExitCode: f.exitCode,
		Error:    f.err,
	}}
}

func (f *fakeParallelCmd) GetLabel() string {
	return f.label
}

func (f *fakeParallelCmd) SetCwd(_ string) {
	// No-op for the fake command
}

func TestParallelBatchRun_AllSuccess(t *testing.T) {
	defer goleak.VerifyNone(t)
	batch := &ParallelBatch{
		Label: "parallel-batch-success",
		Commands: []Runnable{
			&fakeParallelCmd{label: "cmd1", delay: 10 * time.Millisecond, exitCode: 0},
			&fakeParallelCmd{label: "cmd2", delay: 20 * time.Millisecond, exitCode: 0},
		},
	}
	ctx := context.Background()
	results := batch.Run(ctx)
	assert.Len(t, results, 1)
	assert.NoError(t, results[0].Error, "expected no error")
	assert.Len(t, results[0].Children, 2, "expected 2 child results")
	for _, res := range results[0].Children {
		assert.Equal(t, 0, res.ExitCode)
		assert.NoError(t, res.Error)
	}
}

func TestParallelBatchRun_OneFailure(t *testing.T) {
	defer goleak.VerifyNone(t)
	batch := &ParallelBatch{
		Label: "parallel-batch-fail",
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
			assert.Error(t, res.Error)
		}
	}
	assert.True(t, foundFail, "expected at least one failure")
}

func TestParallelBatchRun_Parallelism(t *testing.T) {
	defer goleak.VerifyNone(t)
	batch := &ParallelBatch{
		Label: "parallel-batch-parallelism",
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
