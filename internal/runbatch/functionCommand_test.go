// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFunctionCommandRun_Success(t *testing.T) {
	// Define a function that succeeds
	successFunc := func(_ context.Context, _ string, _ ...string) FunctionCommandReturn {
		return FunctionCommandReturn{}
	}

	cmd := &FunctionCommand{
		BaseCommand: NewBaseCommand("success function", t.TempDir(), RunOnAlways, nil, nil),
		Func:        successFunc,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := cmd.Run(ctx)
	assert.Len(t, results, 1, "expected 1 result")

	res := results[0]
	assert.Equal(t, 0, res.ExitCode, "expected exit code 0")
	assert.NoError(t, res.Error, "unexpected error")
}

func TestFunctionCommandRun_Failure(t *testing.T) {
	// Define a custom error for testing
	testErr := errors.New("function failed") //nolint:err113

	// Define a function that fails with our custom error
	failFunc := func(_ context.Context, _ string, _ ...string) FunctionCommandReturn {
		return FunctionCommandReturn{Err: testErr}
	}

	cmd := &FunctionCommand{
		BaseCommand: NewBaseCommand("failure function", t.TempDir(), RunOnAlways, nil, nil),
		Func:        failFunc,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := cmd.Run(ctx)
	assert.Len(t, results, 1, "expected 1 result")

	res := results[0]
	assert.Equal(t, -1, res.ExitCode, "expected -1 exit code")
	require.Error(t, res.Error, "expected error")
	require.ErrorIs(t, res.Error, testErr, "expected specific error")
}

func TestFunctionCommandRun_NilFunction(t *testing.T) {
	// Test with a nil function
	cmd := &FunctionCommand{
		BaseCommand: NewBaseCommand("nil function", t.TempDir(), RunOnAlways, nil, nil),
		Func:        nil,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This should not panic
	results := cmd.Run(ctx)
	assert.Len(t, results, 1, "expected 1 result")

	res := results[0]
	assert.Equal(t, 0, res.ExitCode, "expected exit code 0")
	assert.NoError(t, res.Error, "unexpected error")
}

func TestFunctionCommandRun_ContextCancelled(t *testing.T) {
	// Define a function that blocks for longer than the context timeout
	longRunningFunc := func(_ context.Context, _ string, _ ...string) FunctionCommandReturn {
		time.Sleep(500 * time.Millisecond)
		return FunctionCommandReturn{}
	}

	cmd := &FunctionCommand{
		BaseCommand: NewBaseCommand("timeout function", t.TempDir(), RunOnAlways, nil, nil),
		Func:        longRunningFunc,
	}

	// Use a short timeout that will be exceeded
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	results := cmd.Run(ctx)
	assert.Len(t, results, 1, "expected 1 result")

	res := results[0]
	assert.Equal(t, -1, res.ExitCode, "expected -1 exit code for cancelled context")
	require.Error(t, res.Error, "expected error for cancelled context")
	require.ErrorIs(t, res.Error, context.DeadlineExceeded, "expected deadline exceeded error")
}

func TestFunctionCommandRun_PanicHandling(t *testing.T) {
	// Define a function that panics
	panicFunc := func(_ context.Context, _ string, _ ...string) FunctionCommandReturn {
		panic("function panicked")
	}

	cmd := &FunctionCommand{
		BaseCommand: NewBaseCommand("panic function", t.TempDir(), RunOnAlways, nil, nil),
		Func:        panicFunc,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// This should not cause the test to panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The function should not propagate panics to the caller: %v", r)
		}
	}()

	_ = cmd.Run(ctx)
}

func TestFunctionCommandRun_Slow(t *testing.T) {
	// Define a slow but eventually succeeding function
	slowFunc := func(_ context.Context, _ string, _ ...string) FunctionCommandReturn {
		time.Sleep(100 * time.Millisecond)
		return FunctionCommandReturn{}
	}

	cmd := &FunctionCommand{
		BaseCommand: NewBaseCommand("slow function", t.TempDir(), RunOnAlways, nil, nil),
		Func:        slowFunc,
	}

	// Use a timeout longer than the function execution time
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	results := cmd.Run(ctx)
	assert.Len(t, results, 1, "expected 1 result")

	res := results[0]
	assert.Equal(t, 0, res.ExitCode, "expected exit code 0")
	assert.NoError(t, res.Error, "unexpected error")
}

func TestFunctionCommandRun_NoGoroutineLeak(t *testing.T) {
	// Define a function that blocks until given channel is closed
	blockCh := make(chan struct{})
	blockingFunc := func(_ context.Context, _ string, _ ...string) FunctionCommandReturn {
		<-blockCh // Block until channel is closed
		return FunctionCommandReturn{}
	}

	cmd := &FunctionCommand{
		BaseCommand: NewBaseCommand("blocking function", t.TempDir(), RunOnAlways, nil, nil),
		Func:        blockingFunc,
	}

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Start the command and immediately cancel the context
	go func() {
		time.Sleep(10 * time.Millisecond) // Give the function time to start
		cancel()
	}()

	results := cmd.Run(ctx)
	assert.Len(t, results, 1, "expected 1 result")

	res := results[0]
	assert.Equal(t, -1, res.ExitCode, "expected -1 exit code for cancelled context")
	require.ErrorIs(t, res.Error, context.Canceled, "expected context cancelled error")

	// Close the channel to allow any leaked goroutine to exit
	close(blockCh)

	// Wait a bit for any goroutines to clean up
	time.Sleep(50 * time.Millisecond)
}
