package runbatch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSerialBatchRun_Integration_AllSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	batch := &SerialBatch{
		Label: "integration-batch-success",
		Commands: []Runnable{
			&OSCommand{Path: "/bin/echo", Args: []string{"hello"}, Label: "echo1"},
			&OSCommand{Path: "/bin/echo", Args: []string{"world"}, Label: "echo2"},
		},
	}
	results := batch.Run(ctx)
	assert.Len(t, results, 1)
	res := results[0]
	assert.Equal(t, 0, res.ExitCode)
	assert.NoError(t, res.Error)
	assert.Len(t, res.Children, 2)
	assert.Contains(t, string(res.Children[0].StdOut), "hello")
	assert.Contains(t, string(res.Children[1].StdOut), "world")
}

func TestSerialBatchRun_Integration_OneFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	batch := &SerialBatch{
		Label: "integration-batch-fail",
		Commands: []Runnable{
			&OSCommand{Path: "/bin/echo", Args: []string{"ok"}, Label: "echo-ok"},
			&OSCommand{Path: "/bin/false", Args: []string{}, Label: "fail-cmd"},
		},
	}
	results := batch.Run(ctx)
	assert.Len(t, results, 1)
	res := results[0]
	assert.NotEqual(t, 0, res.ExitCode)
	assert.Error(t, res.Error)
	assert.Len(t, res.Children, 2)
	assert.Contains(t, string(res.Children[0].StdOut), "ok")
}

func TestSerialBatchRun_Integration_NestedBatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	childBatch := &SerialBatch{
		Label: "child-integration",
		Commands: []Runnable{
			&OSCommand{Path: "/bin/echo", Args: []string{"child"}, Label: "child-echo"},
			&OSCommand{Path: "/bin/sh", Args: []string{"-c", "exit 123"}, Label: "child-fail"},
		},
	}
	batch := &SerialBatch{
		Label: "parent-integration",
		Commands: []Runnable{
			childBatch,
			&OSCommand{Path: "/bin/echo", Args: []string{"parent"}, Label: "parent-echo"},
		},
	}
	results := batch.Run(ctx)
	assert.Len(t, results, 1)
	res := results[0]

	// Check parent batch result
	assert.Equal(t, -1, res.ExitCode, "expected -1 exit code in parent batch")
	assert.ErrorIs(t, res.Error, ErrResultChildrenHasError, "expected error to be ErrResultChildrenHasError")
	assert.Len(t, res.Children, 2)

	// Check child batch result
	assert.Equal(t, "child-integration", res.Children[0].Label)
	assert.ErrorIs(t, res.Children[0].Error, ErrResultChildrenHasError, "expected error to be ErrResultChildrenHasError")

	// Check child batch's child results
	assert.Len(t, res.Children[0].Children, 2)
	assert.Equal(t, "child-echo", res.Children[0].Children[0].Label)
	assert.Equal(t, "child-fail", res.Children[0].Children[1].Label)
	assert.Equal(t, 0, res.Children[0].Children[0].ExitCode)
	assert.Equal(t, 123, res.Children[0].Children[1].ExitCode)

	// Check parent batch's child result
	assert.Equal(t, "parent-echo", res.Children[1].Label)
	assert.Equal(t, 0, res.Children[1].ExitCode)
	assert.NoError(t, res.Children[1].Error)
	assert.Contains(t, string(res.Children[1].StdOut), "parent")
}
